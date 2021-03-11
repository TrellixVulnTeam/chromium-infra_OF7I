-- This query updates the 'P50 Phase Runtime' and 'P90 Phase Runtime' metrics in the cq_builder_metrics_week
-- table. It uses an interval of the last 8 days so that on Mondays, it will update stats for the full
-- previous week as well as stats for the current week.
-- This query is meant to be almost identical to the one in cq_builder_metrics_week_day.sql.

-- The lines below are used by the deploy tool.
--name: Populate cq_builder_metrics_week phase metrics
--schedule: every 8 hours synchronized

DECLARE start_date DATE DEFAULT DATE_SUB(CURRENT_DATE('PST8PDT'), INTERVAL 8 DAY);
-- This isn't really needed, but useful to have around when doing backfills
-- The end_date is exclusive, so if it's on a Monday then that week won't be
-- included. This is why we add a day to it.
DECLARE end_date DATE DEFAULT DATE_ADD(CURRENT_DATE('PST8PDT'), INTERVAL 1 DAY);
DECLARE start_ts TIMESTAMP DEFAULT TIMESTAMP(start_date, 'PST8PDT');
DECLARE end_ts TIMESTAMP DEFAULT TIMESTAMP(end_date, 'PST8PDT');

-- Merge statement
MERGE INTO
  `chrome-trooper-analytics.metrics.cq_builder_metrics_week` AS T
USING
  (
  WITH steps AS (
    SELECT
      id,
      b.builder.builder builder,
      b.start_time,
      cq.chromium_recipe_phase(s.name, st.start_time).*,
      TIMESTAMP_DIFF(s.end_time, s.start_time, SECOND) duration,
      TIMESTAMP_DIFF(s.end_time, st.start_time, SECOND) test_duration
    FROM
      `cr-buildbucket.chromium.builds` b,
      UNNEST(steps) s LEFT OUTER JOIN
      UNNEST(steps) st ON st.name LIKE CONCAT('%|[trigger] ', s.name),
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
    phase_rollup AS (
    SELECT
      DATE_TRUNC(EXTRACT(DATE FROM ANY_VALUE(start_time) AT TIME ZONE "PST8PDT"), WEEK(MONDAY)) AS `date`,
      id,
      ANY_VALUE(builder) builder,
      -- Needed to keep track of last update
      ANY_VALUE(start_time) start_time,
      section,
      phase,
      MAX(CASE phase WHEN 'test' THEN test_duration ELSE duration END) duration
    FROM
      steps
    WHERE
      section = 'patch' AND phase IN ('bot_update','gclient runhooks', 'compile', 'test')
    GROUP BY id, section, phase
    ),
    phase_metrics AS (
    SELECT
      r.date,
      r.builder,
      r.phase,
      MIN(start_time) min_start_time,
      MAX(start_time) max_start_time,
      APPROX_QUANTILES(r.duration, 100) duration
    FROM
      phase_rollup r
    WHERE
      -- This makes sure that we only include full weeks.  For example, if the start_date is a
      -- Wednesday, the week will start on Monday, which will fall outside of the allowed range.
      r.date >= start_date AND r.date < end_date
    GROUP BY
      date, builder, phase
    )
  SELECT
    date,
    'P50 Phase Runtime' AS metric,
    builder,
    max(max_start_time) as max_start_time,
    ARRAY_AGG(
      STRUCT(phase AS label, CAST(duration[OFFSET(50)] AS NUMERIC) AS value)
      ORDER BY phase
    ) AS value_agg,
  FROM phase_metrics
  GROUP BY date, builder
  UNION ALL
  SELECT
    date,
    'P90 Phase Runtime' AS metric,
    builder,
    max(max_start_time) as max_start_time,
    ARRAY_AGG(
      STRUCT(phase AS label, CAST(duration[OFFSET(90)] AS NUMERIC) AS value)
      ORDER BY phase
    ) AS value_agg,
  FROM phase_metrics
  GROUP BY date, builder
  ) S
ON
  T.date = S.date AND T.metric = S.metric AND T.builder = S.builder
  WHEN MATCHED AND T.checkpoint != string(S.max_start_time, "UTC") THEN
    UPDATE SET value_agg = S.value_agg, checkpoint = string(S.max_start_time, "UTC"), last_updated = current_timestamp()
  WHEN NOT MATCHED THEN
    INSERT (date, metric, builder, value_agg, last_updated, checkpoint)
    VALUES (date, metric, builder, value_agg, current_timestamp(), string(max_start_time, "UTC"));
