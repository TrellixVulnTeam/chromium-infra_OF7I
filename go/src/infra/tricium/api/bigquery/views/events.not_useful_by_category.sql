-- Summary of not useful scores by project and analyzer.
SELECT
  gerrit_project,
  category,
  SUM(not_useful_count) AS sum_not_useful_count,
  SUM(total_count) AS sum_total_count,
  SUM(not_useful_count) / SUM(total_count) AS proportion
FROM `tricium-prod.events.not_useful_proportion`
GROUP BY
  gerrit_project,
  category
ORDER BY
  gerrit_project,
  category;
