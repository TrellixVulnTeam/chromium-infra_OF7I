-- This query updates the 'P50 Phase Runtime' and 'P90 Phase Runtime' metrics in the cq_builder_metrics_day
-- table. It uses an interval of the last 2 days so that there is some redundancy if the job fails
-- This query is meant to be almost identical to the one in cq_builder_metrics_week_phase.sql.

-- The lines below are used by the deploy tool.
--name: Populate cq_builder_metrics_day phase metrics
--schedule: every 4 hours synchronized

DECLARE start_date DATE DEFAULT DATE_SUB(CURRENT_DATE('PST8PDT'), INTERVAL 2 DAY);
-- This isn't really needed, but useful to have around when doing backfills
-- The end_date is exclusive, so if it's on a Monday then that week won't be
-- included. This is why we add a day to it.
DECLARE end_date DATE DEFAULT DATE_ADD(CURRENT_DATE('PST8PDT'), INTERVAL 1 DAY);
DECLARE start_ts TIMESTAMP DEFAULT TIMESTAMP(start_date, 'PST8PDT');
DECLARE end_ts TIMESTAMP DEFAULT TIMESTAMP(end_date, 'PST8PDT');

-- Merge statement
MERGE INTO
  `chrome-trooper-analytics.metrics.cq_builder_metrics_day` AS T
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
      -- As we bucket the build using start_date, we need to include any builds
      -- that were created on the previous day.
      b.create_time >= TIMESTAMP_SUB(start_ts, INTERVAL 1 DAY)
      AND b.create_time < end_ts
      AND b.builder.bucket = 'try'
      AND b.builder.project = 'chromium'
      AND b.builder.builder = cq.builder
    ),
    phase_rollup AS (
    SELECT
      EXTRACT(DATE FROM ANY_VALUE(start_time) AT TIME ZONE "PST8PDT") AS `date`,
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
