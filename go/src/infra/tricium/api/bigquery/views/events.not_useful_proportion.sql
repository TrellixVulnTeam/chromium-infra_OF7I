-- Proportion of comments marked "not useful" by analyzer by date.
WITH
  NotUsefulCounts AS (
  SELECT
    DATE(time) AS report_date,
    gerrit_project,
    analyzer,
    category,
    COUNT(*) AS not_useful_count
  FROM
    `tricium-prod.events.not_useful_comments`
  GROUP BY
    report_date,
    gerrit_project,
    analyzer,
    category),
  TotalCounts AS (
  SELECT
    DATE(time) AS report_date,
    gerrit_project,
    analyzer,
    category,
    COUNT(*) AS total_count
  FROM
    `tricium-prod.events.comment_events`
  WHERE
    event_type = 'COMMENT_POST'
  GROUP BY
    report_date,
    gerrit_project,
    analyzer,
    category)
SELECT
  TotalCounts.report_date,
  TotalCounts.gerrit_project,
  TotalCounts.analyzer,
  TotalCounts.category,
  NotUsefulCounts.not_useful_count,
  TotalCounts.total_count,
  NotUsefulCounts.not_useful_count / TotalCounts.total_count AS proportion
FROM
  NotUsefulCounts
INNER JOIN
  TotalCounts
ON
  NotUsefulCounts.report_date = TotalCounts.report_date
  AND NotUsefulCounts.gerrit_project = TotalCounts.gerrit_project
  AND NotUsefulCounts.analyzer = TotalCounts.analyzer
  AND NotUsefulCounts.category = TotalCounts.category
ORDER BY
  TotalCounts.report_date DESC;
