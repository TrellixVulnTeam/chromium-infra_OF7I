package analysis

const clusterSummariesAnalysis = `
CREATE TEMP FUNCTION sub_cluster(VALUES ARRAY<STRING>) AS (
	(
	  SELECT
		ARRAY_AGG( (SELECT AS STRUCT value, num_fails))
	  FROM (
		SELECT
		  v value,
		  COUNT(*) num_fails
		FROM UNNEST(VALUES) v
		GROUP BY v
		ORDER BY num_fails DESC
	  )
	)
  );

  WITH clustered_failures_latest AS (
	SELECT
	  cluster_algorithm,
	  cluster_id,
	  test_result_system,
	  test_result_id,
	  DATE(partition_time) as partition_time,
	  ARRAY_AGG(cf ORDER BY last_updated DESC LIMIT 1)[OFFSET(0)] as r
	FROM clustered_failures cf
	WHERE partition_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)
	GROUP BY cluster_algorithm, cluster_id, test_result_system, test_result_id, DATE(partition_time)
  ),
  clustered_failures_extended AS (
	SELECT
	  cluster_algorithm,
	  cluster_id,
	  r.is_included,
	  r.is_included_with_high_priority,
	  r.is_exonerated,
	  r.test_id,
	  r.failure_reason,
	  r.test_run_id,
	  r.partition_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 DAY) as is_1d,
	  r.partition_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 3 DAY) as is_3d,
	  r.partition_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY) as is_7d,
	  CONCAT(r.presubmit_run_id.system, ":", r.presubmit_run_id.id) AS presubmit_run_uniqifier,
	  (r.presubmit_run_id IS NOT NULL AND r.is_ingested_invocation_blocked AND
	   r.ingested_invocation_result_index + 1 = r.ingested_invocation_result_count) as is_presubmit_reject,
	  (r.test_run_result_index + 1 = r.test_run_result_count) AND r.is_test_run_blocked as is_test_run_fail,
	FROM clustered_failures_latest
  )

  SELECT
	  cluster_algorithm,
	  cluster_id,

	  -- 1 day metrics.
	  COUNT(DISTINCT IF(is_1d AND is_presubmit_reject AND NOT is_exonerated, presubmit_run_uniqifier, NULL)) as presubmit_rejects_1d,
	  COUNT(DISTINCT IF(is_1d AND is_presubmit_reject, presubmit_run_uniqifier, NULL)) as presubmit_rejects_pre_exon_1d,
	  COUNT(DISTINCT IF(is_1d AND is_presubmit_reject AND is_included_with_high_priority AND NOT is_exonerated, presubmit_run_uniqifier, NULL)) as presubmit_rejects_residual_1d,
	  COUNT(DISTINCT IF(is_1d AND is_presubmit_reject AND is_included_with_high_priority, presubmit_run_uniqifier, NULL)) as presubmit_rejects_residual_pre_exon_1d,
	  COUNT(DISTINCT IF(is_1d AND is_test_run_fail AND NOT is_exonerated, test_run_id, NULL)) as test_run_fails_1d,
	  COUNT(DISTINCT IF(is_1d AND is_test_run_fail, test_run_id, NULL)) as  test_run_fails_pre_exon_1d,
	  COUNT(DISTINCT IF(is_1d AND is_test_run_fail AND is_included_with_high_priority AND NOT is_exonerated, test_run_id, NULL)) as test_run_fails_residual_1d,
	  COUNT(DISTINCT IF(is_1d AND is_test_run_fail AND is_included_with_high_priority, test_run_id, NULL)) as  test_run_fails_residual_pre_exon_1d,
	  COUNTIF(is_1d AND NOT is_exonerated) as failures_1d,
	  COUNTIF(is_1d) AS failures_pre_exon_1d,
	  COUNTIF(is_1d AND is_included_with_high_priority AND NOT is_exonerated) as failures_residual_1d,
	  COUNTIF(is_1d AND is_included_with_high_priority) as failures_residual_pre_exon_1d,
	  sub_cluster(ARRAY_AGG(IF(is_1d, test_id, NULL) IGNORE NULLS)) AS affected_tests_1d,

	  -- 3 day metrics.
	  COUNT(DISTINCT IF(is_3d AND is_presubmit_reject AND NOT is_exonerated, presubmit_run_uniqifier, NULL)) as presubmit_rejects_3d,
	  COUNT(DISTINCT IF(is_3d AND is_presubmit_reject, presubmit_run_uniqifier, NULL)) as presubmit_rejects_pre_exon_3d,
	  COUNT(DISTINCT IF(is_3d AND is_presubmit_reject AND is_included_with_high_priority AND NOT is_exonerated, presubmit_run_uniqifier, NULL)) as presubmit_rejects_residual_3d,
	  COUNT(DISTINCT IF(is_3d AND is_presubmit_reject AND is_included_with_high_priority, presubmit_run_uniqifier, NULL)) as presubmit_rejects_residual_pre_exon_3d,
	  COUNT(DISTINCT IF(is_3d AND is_test_run_fail AND NOT is_exonerated, test_run_id, NULL)) as test_run_fails_3d,
	  COUNT(DISTINCT IF(is_3d AND is_test_run_fail, test_run_id, NULL)) as  test_run_fails_pre_exon_3d,
	  COUNT(DISTINCT IF(is_3d AND is_test_run_fail AND is_included_with_high_priority AND NOT is_exonerated, test_run_id, NULL)) as test_run_fails_residual_3d,
	  COUNT(DISTINCT IF(is_3d AND is_test_run_fail AND is_included_with_high_priority, test_run_id, NULL)) as  test_run_fails_residual_pre_exon_3d,
	  COUNTIF(is_3d AND NOT is_exonerated) as failures_3d,
	  COUNTIF(is_3d) AS failures_pre_exon_3d,
	  COUNTIF(is_3d AND is_included_with_high_priority AND NOT is_exonerated) as failures_residual_3d,
	  COUNTIF(is_3d AND is_included_with_high_priority) as failures_residual_pre_exon_3d,
	  sub_cluster(ARRAY_AGG(IF(is_3d, test_id, NULL) IGNORE NULLS)) AS affected_tests_3d,

	  -- 7 day metrics.
	  COUNT(DISTINCT IF(is_7d AND is_presubmit_reject AND NOT is_exonerated, presubmit_run_uniqifier, NULL)) as presubmit_rejects_7d,
	  COUNT(DISTINCT IF(is_7d AND is_presubmit_reject, presubmit_run_uniqifier, NULL)) as presubmit_rejects_pre_exon_7d,
	  COUNT(DISTINCT IF(is_7d AND is_presubmit_reject AND is_included_with_high_priority AND NOT is_exonerated, presubmit_run_uniqifier, NULL)) as presubmit_rejects_residual_7d,
	  COUNT(DISTINCT IF(is_7d AND is_presubmit_reject AND is_included_with_high_priority, presubmit_run_uniqifier, NULL)) as presubmit_rejects_residual_pre_exon_7d,
	  COUNT(DISTINCT IF(is_7d AND is_test_run_fail AND NOT is_exonerated, test_run_id, NULL)) as test_run_fails_7d,
	  COUNT(DISTINCT IF(is_7d AND is_test_run_fail, test_run_id, NULL)) as  test_run_fails_pre_exon_7d,
	  COUNT(DISTINCT IF(is_7d AND is_test_run_fail AND is_included_with_high_priority AND NOT is_exonerated, test_run_id, NULL)) as test_run_fails_residual_7d,
	  COUNT(DISTINCT IF(is_7d AND is_test_run_fail AND is_included_with_high_priority, test_run_id, NULL)) as  test_run_fails_residual_pre_exon_7d,
	  COUNTIF(is_7d AND NOT is_exonerated) as failures_7d,
	  COUNTIF(is_7d) AS failures_pre_exon_7d,
	  COUNTIF(is_7d AND is_included_with_high_priority AND NOT is_exonerated) as failures_residual_7d,
	  COUNTIF(is_7d AND is_included_with_high_priority) as failures_residual_pre_exon_7d,
	  sub_cluster(ARRAY_AGG(IF(is_7d, test_id, NULL) IGNORE NULLS)) AS affected_tests_7d,

	  ANY_VALUE(failure_reason) as example_failure_reason,
	  MIN(test_id) as example_test_id,
  FROM clustered_failures_extended
  WHERE is_included
  GROUP BY cluster_algorithm, cluster_id`
