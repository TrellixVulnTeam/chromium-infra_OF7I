CREATE OR REPLACE VIEW `APP_ID.PROJECT_NAME.failing_steps`
AS
/*
Failing steps table.
Each row represents a step that has failed in the most recent run of the
given builder (bucket, project etc).
As the status of the build system changes, so should the contents of this
view.
*/
WITH
  latest_builds AS (
  SELECT
    b.builder.project,
    b.builder.bucket,
    b.builder.builder,
    JSON_EXTRACT_SCALAR(input.properties, "$.builder_group") as buildergroup,
    ARRAY_AGG(b
    ORDER BY
      # Latest, meaning sort by commit position if it exists, otherwise by the build id or number.
      b.output.gitiles_commit.position DESC, id, number DESC
    LIMIT
      1)[
  OFFSET
    (0)] latest
  FROM
    `cr-buildbucket.raw.completed_builds_prod` AS b
  WHERE
    PROJECT_FILTER_CONDITIONS
  GROUP BY
    1,
    2,
    3,
    4),
  recent_tests AS (
    SELECT
      SUBSTR(exported.id, 7) as build_id,
      (SELECT value FROM UNNEST((ARRAY_CONCAT(tags, parent.tags))) WHERE key = "step_name" limit 1) as step_name,
      (SELECT value FROM UNNEST(tags) WHERE key = "test_name") as test_name,
    FROM
      `RESULTDB_DATASET.ci_test_results`
    WHERE
      partition_time > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 DAY)
    GROUP BY
      build_id,
      step_name,
      test_name
    HAVING
      # We do not care about unexpectedly passed tests.
      LOGICAL_AND(not expected)
      AND LOGICAL_AND(status != 'PASS')
  )
SELECT
  project,
  bucket,
  builder,
  latest.number,
  b.latest.id build_id,
  JSON_EXTRACT_SCALAR(latest.input.properties, "$.builder_group") as buildergroup,
  s.name step,
  ANY_VALUE(JSON_EXTRACT_STRING_ARRAY(b.latest.input.properties, "$.sheriff_rotations")) as sheriff_rotations,
  ANY_VALUE(b.latest.status) status,
  ANY_VALUE(b.latest.critical) critical,
  ANY_VALUE(b.latest.output.gitiles_commit) output_commit,
  ANY_VALUE(b.latest.input.gitiles_commit) input_commit,
  FARM_FINGERPRINT(STRING_AGG(tr.test_name, "\n"
    ORDER BY
      tr.test_name)) AS test_names_fp,
  STRING_AGG(tr.test_name, "\n"
  ORDER BY
    tr.test_name
  LIMIT
    40) AS test_names_trunc,
  COUNT(tr.test_name) AS num_tests
FROM
  latest_builds b,
  b.latest.steps s
LEFT OUTER JOIN
  recent_tests tr
ON
  SAFE_CAST(tr.build_id AS int64) = b.latest.id
  AND tr.step_name = s.name
WHERE
  (b.latest.status = 'FAILURE' AND s.status = 'FAILURE')
  OR
  (
    b.latest.status = 'INFRA_FAILURE'
    AND (s.status = 'INFRA_FAILURE' OR s.status = 'CANCELED')
  )
GROUP BY
  1,
  2,
  3,
  4,
  5,
  6,
  7
