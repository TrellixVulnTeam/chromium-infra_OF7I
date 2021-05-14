-- This query updates the 'P50 Slow Tests' and 'P90 Slow Tests' metrics in the cq_builder_metrics_week
-- table. It uses an interval of the last 8 days so that on Mondays, it will update stats for the full
-- previous week as well as stats for the current week.
-- Slow tests are defined as tests that take longer than 5 minutes to run at P50, or 10m to run at P90.
-- We also exclude any tests that have fewer than 50 runs to remove one-off test runs like
-- https://crrev.com/c/2545074/13
-- This query is meant to be almost identical to the one in cq_builder_metrics_day_slow_tests.sql.

-- The lines below are used by the deploy tool.
--name: Populate cq_builder_metrics_week slow test metrics
--schedule: every 8 hours synchronized

DECLARE start_date DATE DEFAULT DATE_SUB(CURRENT_DATE('PST8PDT'), INTERVAL 8 DAY);
-- This isn't really needed, but useful to have around when doing backfills
-- The end_date is exclusive, so if it's on a Monday then that week won't be
-- included. This is why we add a day to it.
DECLARE end_date DATE DEFAULT DATE_ADD(CURRENT_DATE('PST8PDT'), INTERVAL 1 DAY);
DECLARE start_ts TIMESTAMP DEFAULT TIMESTAMP(start_date, 'PST8PDT');
DECLARE end_ts TIMESTAMP DEFAULT TIMESTAMP(end_date, 'PST8PDT');
DECLARE slow_p50_cutoff INT64 DEFAULT 300;
DECLARE slow_p90_cutoff INT64 DEFAULT 600;

-- Merge statement
MERGE INTO
  `chrome-trooper-analytics.metrics.cq_builder_metrics_week` AS T
USING
  (
  WITH builds AS (
    SELECT
      DATE_TRUNC(EXTRACT(DATE FROM b.start_time AT TIME ZONE "PST8PDT"), WEEK(MONDAY)) AS `date`,
      b.id,
      b.builder.builder,
      b.start_time,
      b.infra.swarming.task_id
    FROM
      `cr-buildbucket.chromium.builds` b,
      `chrome-trooper-analytics.metrics.cq_builders` cq
    WHERE
      -- Include builds that were created on Sunday but started on Monday
      b.create_time >= TIMESTAMP_SUB(start_ts, INTERVAL 1 DAY)
      -- If the end date is on a Tuesday, make sure to include all builds for that week.
      AND b.create_time < TIMESTAMP_ADD(end_ts, INTERVAL 8 DAY)
      AND b.builder.bucket = 'try'
      AND b.builder.project = 'chromium'
      AND b.builder.builder = cq.builder
    ),
    tests AS (
    SELECT
      b.id,
      ANY_VALUE(b.date) date,
      ANY_VALUE(b.builder) builder,
      ANY_VALUE(b.start_time) builder_start_time,
      (
        SELECT SUBSTR(i, 6) FROM t.request.tags i WHERE i LIKE 'name:%'
      ) test_name,
      MAX(TIMESTAMP_DIFF(t.start_time, t.create_time, SECOND)) max_shard_pending,
      MAX(TIMESTAMP_DIFF(t.end_time, t.start_time, SECOND)) max_shard_runtime
    FROM
      `chromium-swarm.swarming.task_results_summary` t,
      builds b
    WHERE
      -- This makes sure that we only include full weeks.  For example, if the start_date is a
      -- Wednesday, the week will start on Monday, which will fall outside of the allowed range.
      b.date >= start_date AND b.date < end_date
      AND t.end_time >= TIMESTAMP_SUB(start_ts, INTERVAL 1 DAY)
      AND t.end_time < TIMESTAMP_ADD(end_ts, INTERVAL 1 DAY)
      AND t.request.parent_task_id = b.task_id
    GROUP BY id, test_name
    ),
    tests_grouped AS (
    SELECT
      t.date,
      t.builder,
      MIN(t.builder_start_time) min_builder_start_time,
      MAX(t.builder_start_time) max_builder_start_time,
      t.test_name,
      APPROX_QUANTILES(t.max_shard_pending, 100) max_shard_pending,
      APPROX_QUANTILES(t.max_shard_runtime, 100) max_shard_runtime,
      count(id) num_runs,
      countif(t.max_shard_runtime > slow_p50_cutoff) num_slow_runs,
    FROM tests t
    GROUP BY t.date, t.builder, t.test_name
    ),
    slow_tests AS (
    SELECT *
    FROM tests_grouped
    WHERE (max_shard_runtime[OFFSET(50)] > slow_p50_cutoff or max_shard_runtime[OFFSET(90)] > slow_p90_cutoff)
      AND num_runs > 50
    )
  SELECT
    date,
    'P50 Slow Tests' AS metric,
    builder,
    MAX(max_builder_start_time) max_builder_start_time,
    ARRAY_AGG(
      STRUCT(test_name AS label, CAST(max_shard_runtime[OFFSET(50)] AS NUMERIC) AS value)
      ORDER BY test_name
    ) AS value_agg,
  FROM slow_tests
  GROUP BY date, builder
  UNION ALL
  SELECT
    date,
    'P90 Slow Tests' AS metric,
    builder,
    MAX(max_builder_start_time) AS max_builder_start_time,
    ARRAY_AGG(
      STRUCT(test_name AS label, CAST(max_shard_runtime[OFFSET(90)] AS NUMERIC) AS value)
      ORDER BY test_name
    ) AS value_agg,
  FROM slow_tests
  GROUP BY date, builder
  UNION ALL
  SELECT
    date,
    'Count Slow Tests' AS metric,
    builder,
    MAX(max_builder_start_time) AS max_builder_start_time,
    ARRAY_AGG(
      STRUCT(test_name AS label, CAST(num_slow_runs AS NUMERIC) AS value)
      ORDER BY test_name
    ) AS value_agg,
  FROM slow_tests
  GROUP BY date, builder
   ) S
ON
  T.date = S.date AND T.metric = S.metric AND T.builder = S.builder
  WHEN MATCHED AND T.checkpoint != string(S.max_builder_start_time, "UTC") THEN
    UPDATE SET value_agg = S.value_agg, checkpoint = string(S.max_builder_start_time, "UTC"), last_updated = current_timestamp()
  WHEN NOT MATCHED THEN
    INSERT (date, metric, builder, value_agg, last_updated, checkpoint)
    VALUES (date, metric, builder, value_agg, current_timestamp(), string(max_builder_start_time, "UTC"));
