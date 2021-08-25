-- This query keeps the cq_builders table up-to-date. The table holds a list
-- of all the main CQ builders. This scheduled query makes sure that when a new
-- builder is added to CQ, it is also added to the cq_builders table.

-- The lines below are used by the deploy tool.
--name: Populate cq_builders
--schedule: every 24 hours synchronized
DECLARE start_ts TIMESTAMP DEFAULT TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY);

MERGE INTO
  `chrome-trooper-analytics.metrics.cq_builders` AS T
USING
  (
  WITH builds AS (
    SELECT
      b.builder.builder,
      COUNT(*) num_builds
    FROM
      `cr-buildbucket.chromium.builds` b
    WHERE
      b.create_time >= start_ts
      AND b.builder.bucket = 'try'
      AND b.builder.project = 'chromium'
    GROUP BY b.builder.builder
  )
  SELECT
    builder
  FROM
    builds
  WHERE
    -- Include the builder if it's run at least 50% of the time.
    num_builds >= (SELECT MAX(num_builds) / 2 FROM builds)
    -- Also, filter out any builder if they're not run at least 3k times in a
    -- week. This is meant to avoid adding a bunch of opt-in builders on very
    -- slow CQ weeks like Christmas/New Years
    AND num_builds >= 3000
  ) AS S
ON
  T.builder = S.builder
WHEN NOT MATCHED THEN
  INSERT (builder) VALUES (builder)
