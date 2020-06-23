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
    JSON_EXTRACT_SCALAR(input.properties,
      "$.mastername") AS mastername,
    ARRAY_AGG(b
    ORDER BY
      # Latest, meaning sort by commit position if it exists, otherwise by the build id or number.
      b.output.gitiles_commit.position DESC, id, number DESC
    LIMIT
      1)[
  OFFSET
    (0)] latest
  FROM
    `cr-buildbucket.PROJECT_NAME.builds` AS b
  WHERE
    create_time > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)
  GROUP BY
    1,
    2,
    3,
    4)
SELECT
  project,
  bucket,
  builder,
  latest.number,
  b.latest.id build_id,
  JSON_EXTRACT_SCALAR(latest.input.properties,
    "$.mastername") AS mastername,
  s.name step,
  ANY_VALUE(b.latest.status) status,
  ANY_VALUE(b.latest.critical) critical,
  ANY_VALUE(b.latest.output.gitiles_commit) output_commit,
  ANY_VALUE(b.latest.input.gitiles_commit) input_commit,
  NULL AS test_names_fp,
  CAST(NULL as STRING) AS test_names_trunc,
  0 AS num_tests
FROM
  latest_builds b,
  b.latest.steps s
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
