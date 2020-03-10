-- Number of selected comments each day per project, analyzer category, etc.
WITH SelectedComments AS (
  SELECT
    requested_time,
    gerrit_revision.project AS gerrit_project,
    c.analyzer,
    c.comment.category,
  FROM
    `tricium-prod.analyzer.results`,
    UNNEST(comments) AS c
  WHERE
    -- Selected comments are those comments that are actually posted.
    c.selected IS TRUE)
SELECT
  DATE(requested_time) AS requested_date,
  gerrit_project,
  analyzer,
  category,
  COUNT(*) AS num_comments
FROM SelectedComments
GROUP BY
  requested_date,
  gerrit_project,
  analyzer,
  category
ORDER BY
  requested_date DESC,
  gerrit_project ASC,
  category ASC;
