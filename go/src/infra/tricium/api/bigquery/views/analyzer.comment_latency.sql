-- One row per day per analyzer, with average comment latency, which is the
-- difference between analyze request time and comment creation time.
--
-- This latency is approximately how long it took for the analyzer to run.
WITH
  CommentPostings AS (
  SELECT
    c.analyzer,
    requested_time,
    c.created_time,
    TIMESTAMP_DIFF(created_time, requested_time, SECOND) AS latency_seconds,
  FROM
    `tricium-prod.analyzer.results`,
    UNNEST(comments) AS c)
SELECT
  DATE(requested_time) AS request_date,
  analyzer,
  AVG(latency_seconds) AS avg_latency_seconds
FROM
  CommentPostings
GROUP BY
  analyzer,
  request_date
ORDER BY
  request_date DESC,
  analyzer ASC;
