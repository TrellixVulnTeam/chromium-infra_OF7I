-- One row per comment posted on the final patchset of a change.
--
-- These could be thought of as roughly the "ignored" comments, that were never
-- fixed and thus probably not useful.
WITH
  -- LastRevisions has one row for each change and includes
  -- the final revision (patchset) number.
  LastRevisions AS (
  SELECT
    gerrit_revision.change,
    MAX(revision_number) AS last_revision_number
  FROM
    `tricium-prod.analyzer.results`
  GROUP BY
    change),
  -- LastRevisionSelectedComments has one row per comment that is
  -- posted on the final revision of a change.
  LastRevisionSelectedComments AS (
  SELECT
    requested_time,
    gerrit_revision.project AS gerrit_project,
    LastRevisions.last_revision_number,
    c.analyzer,
    c.comment.category
  FROM
    `tricium-prod.analyzer.results`,
    UNNEST(comments) AS c
  JOIN
    LastRevisions
  ON
    gerrit_revision.change = LastRevisions.change
    AND revision_number = LastRevisions.last_revision_number
  WHERE
    -- Selected comments are those comments that are actually posted.
    c.selected = TRUE)
SELECT
  DATE(requested_time) AS requested_date,
  gerrit_project,
  analyzer,
  category,
  COUNT(*) AS num_comments
FROM LastRevisionSelectedComments
GROUP BY
  requested_date,
  gerrit_project,
  analyzer,
  category
ORDER BY
  requested_date DESC,
  gerrit_project ASC,
  category ASC;
