-- This view has one row per comment that has a not useful report,
-- and contains comment details.
SELECT DISTINCT
  time,
  comment_id,
  gerrit_project,
  change,
  event_type,
  analyzer,
  category,
  message
FROM
  `tricium-prod.events.comment_events`
WHERE
  event_type = 'NOT_USEFUL'
ORDER BY
  time DESC;
