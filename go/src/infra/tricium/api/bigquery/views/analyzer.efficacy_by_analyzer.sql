-- Summary of efficacy for each analyzer over all time.
SELECT
  gerrit_project,
  analyzer,
  SUM(total_comments) AS sum_total_comments,
  SUM(last_comments) AS sum_last_comments,
  1 - (SUM(last_comments) / SUM(total_comments)) AS efficacy
FROM
  `tricium-prod.analyzer.efficacy`
GROUP BY
  gerrit_project,
  analyzer
ORDER BY
  gerrit_project,
  analyzer;
