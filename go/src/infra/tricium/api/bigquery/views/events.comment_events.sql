-- This view contains one row per event (comment post or not useful report)
-- and contains extra information about the change ID and comment details.
SELECT
  time,
  FlattenedComments.id AS comment_id,
  CommentIDs.gerrit_project,
  CommentIDs.change,
  revision_number,
  type AS event_type,
  SPLIT(FlattenedComments.category, '/')[OFFSET(0)] AS analyzer,
  FlattenedComments.category AS category,
  FlattenedComments.message AS message
FROM
  `tricium-prod.events.feedback`
CROSS JOIN
  UNNEST(comments) AS FlattenedComments
INNER JOIN
  `tricium-prod.analyzer.comments_with_ids` AS CommentIDs
ON
  FlattenedComments.id = CommentIDs.comment_id
ORDER BY
  time DESC;
