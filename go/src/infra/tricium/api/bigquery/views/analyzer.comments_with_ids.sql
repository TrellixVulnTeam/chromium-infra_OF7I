-- Comments, including both change ID and comment IDs.
--
-- This can be used in joins with the events table,
-- which also includes comment IDs.
SELECT
  gerrit_revision.project AS gerrit_project,
  gerrit_revision.change,
  revision_number,
  FlattenedComments.comment.id AS comment_id,
  FlattenedComments.analyzer AS analyzer,
  FlattenedComments.comment.category AS category,
  FlattenedComments.comment.path AS comment_path
FROM
  `tricium-prod.analyzer.results`
CROSS JOIN
  UNNEST(comments) AS FlattenedComments;
