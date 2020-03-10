-- Proportion of comments that were "fixed" for each analyzer by date.
--
-- "Efficacy" for Tricium analyzers is a proxy for how useful the analyzer's
-- comments are; it is the proportion of the total comments that were posted
-- that were *not* posted on the final patchset.
SELECT
  TotalComments.requested_date,
  TotalComments.gerrit_project,
  TotalComments.analyzer,
  TotalComments.category,
  TotalComments.num_comments AS total_comments,
  LastComments.num_comments AS last_comments,
  1 - (LastComments.num_comments / TotalComments.num_comments) AS efficacy
FROM
  `tricium-prod.analyzer.comments_selected` AS TotalComments
JOIN
  `tricium-prod.analyzer.comments_final` AS LastComments
ON
  TotalComments.requested_date = LastComments.requested_date
  AND TotalComments.gerrit_project = LastComments.gerrit_project
  AND TotalComments.analyzer = LastComments.analyzer
  AND TotalComments.category = LastComments.category
ORDER BY
  requested_date DESC;
