-- This query updates most of the metrics in the cq_builder_metrics_day table, except for the phase
-- ones. It uses an interval of the last 2 days so that there is some redundancy if the job fails
-- This query is meant to be almost identical to the one in cq_builder_metrics_week.sql.

-- The lines below are used by the deploy tool.
--name: Populate cq_builder_metrics_day
--schedule: every 4 hours synchronized

DECLARE start_date DATE DEFAULT DATE_SUB(CURRENT_DATE('PST8PDT'), INTERVAL 2 DAY);
-- This isn't really needed, but useful to have around when doing backfills
-- The end_date is exclusive, which is why we add a day to the current date.
DECLARE end_date DATE DEFAULT DATE_ADD(CURRENT_DATE('PST8PDT'), INTERVAL 1 DAY);
DECLARE start_ts TIMESTAMP DEFAULT TIMESTAMP(start_date, 'PST8PDT');
DECLARE end_ts TIMESTAMP DEFAULT TIMESTAMP(end_date, 'PST8PDT');

-- Merge statement
MERGE INTO
  `chrome-trooper-analytics.metrics.cq_builder_metrics_day` AS T
USING
  (
  WITH stats AS (
    SELECT
      EXTRACT(DATE FROM b.start_time AT TIME ZONE "PST8PDT") AS `date`,
      b.start_time,
      b.builder.project AS project,
      b.builder.builder AS builder,
      TIMESTAMP_DIFF(b.end_time, b.create_time, SECOND) total,
      TIMESTAMP_DIFF(b.end_time, b.start_time, SECOND) runtime,
      TIMESTAMP_DIFF(b.start_time, b.create_time, SECOND) pending
    FROM
      `cr-buildbucket.chromium.builds` b,
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
    stats_grouped AS (
    SELECT
      d.date,
      -- To inspect date boundaries
      MIN(d.start_time) min_start_time,
      -- To inspect date boundaries
      MAX(d.start_time) max_start_time,
      d.builder,
      COUNT(*) cnt,
      APPROX_QUANTILES(total, 100) total,
      APPROX_QUANTILES(runtime, 100) runtime,
      APPROX_QUANTILES(pending, 100) pending
    FROM
      stats d
    WHERE
      d.date >= start_date AND d.date < end_date
    GROUP BY
      d.date, d.builder
  )
  SELECT date, 'P50' AS metric, builder, max_start_time, total[OFFSET(50)] AS value FROM stats_grouped
  UNION ALL SELECT date, 'P90' AS metric, builder, max_start_time, total[OFFSET(90)] AS value FROM stats_grouped
  UNION ALL SELECT date, 'P50 Runtime' AS metric, builder, max_start_time, runtime[OFFSET(50)] AS value FROM stats_grouped
  UNION ALL SELECT date, 'P90 Runtime' AS metric, builder, max_start_time, runtime[OFFSET(90)] AS value FROM stats_grouped
  UNION ALL SELECT date, 'P50 Pending' AS metric, builder, max_start_time, pending[OFFSET(50)] AS value FROM stats_grouped
  UNION ALL SELECT date, 'P90 Pending' AS metric, builder, max_start_time, pending[OFFSET(90)] AS value FROM stats_grouped
  UNION ALL SELECT date, 'Count' AS metric, builder, max_start_time, cnt AS value FROM stats_grouped
  ) S
ON
  T.date = S.date AND T.metric = S.metric AND T.builder = S.builder
  WHEN MATCHED AND T.checkpoint != string(S.max_start_time, "UTC") THEN
    UPDATE SET value = S.value, checkpoint = string(S.max_start_time, "UTC"), last_updated = current_timestamp()
  WHEN NOT MATCHED THEN
    INSERT (date, metric, builder, value, last_updated, checkpoint)
    VALUES (date, metric, builder, value, current_timestamp(), string(max_start_time, "UTC"));
