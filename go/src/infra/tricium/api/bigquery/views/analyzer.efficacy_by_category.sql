-- Summary of efficacy for each category over all time.
--
-- This view may be useful for deciding how to tweak enabled
-- categories or analyzers to avoid noise.
SELECT
  gerrit_project,
  category,
  SUM(total_comments) AS sum_total_comments,
  SUM(last_comments) AS sum_last_comments,
  1 - (SUM(last_comments) / SUM(total_comments)) AS efficacy
FROM
  `tricium-prod.analyzer.efficacy`
GROUP BY
  gerrit_project,
  category
ORDER BY
  gerrit_project,
  category;
