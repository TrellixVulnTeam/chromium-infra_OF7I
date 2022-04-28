package analysis

const clusterPresubmitAnalysis = `
  WITH clustered_failures_latest AS (
	SELECT
	  cluster_algorithm,
	  cluster_id,
	  test_result_system,
	  test_result_id,
	  DATE(partition_time) as partition_time,
	  ARRAY_AGG(cf ORDER BY last_updated DESC LIMIT 1)[OFFSET(0)] as r
	FROM clustered_failures cf
	WHERE partition_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 DAY)
	GROUP BY cluster_algorithm, cluster_id, test_result_system, test_result_id, DATE(partition_time)
  ),
  clustered_failures_extended AS (
	SELECT
	  cluster_algorithm,
	  cluster_id,
	  r.is_included,
	  r.partition_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 12 HOUR) as is_12h,
	  r.partition_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 DAY) as is_1d,
	  -- The identity of the first changelist that was tested, assuming the
	  -- result was part of a presubmit run, and the owner of the presubmit
	  -- run was a user and not automation.
	  IF(ARRAY_LENGTH(r.presubmit_run_cls)>0 AND r.presubmit_run_owner='user',
		  CONCAT(r.presubmit_run_cls[OFFSET(0)].host, r.presubmit_run_cls[OFFSET(0)].change),
		  NULL) as presubmit_run_user_cl_id,
	  r.is_test_run_blocked as is_test_run_fail,
	FROM clustered_failures_latest
  )
  SELECT
    STRUCT(cluster_algorithm AS Algorithm, cluster_id as ID) as ClusterID,
	COUNT(DISTINCT IF(is_12h AND is_test_run_fail, presubmit_run_user_cl_id, NULL)) as DistinctUserClTestRunsFailed12h,
	COUNT(DISTINCT IF(is_1d AND is_test_run_fail, presubmit_run_user_cl_id, NULL)) as DistinctUserClTestRunsFailed1d,
  FROM clustered_failures_extended
  WHERE STRUCT(cluster_algorithm AS Algorithm, cluster_id as ID) IN UNNEST(@clusterIDs)
    AND is_included
  GROUP BY cluster_algorithm, cluster_id
`

const clusterSummariesAnalysis = `
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
	  NOT (r.exoneration_status = 'NOT_EXONERATED') as is_exonerated,
	  NOT (r.exoneration_status = 'NOT_EXONERATED' OR r.exoneration_status = 'WEETBIX') as is_exonerated_pre_weetbix, 
	  r.test_id,
	  r.failure_reason,
	  r.test_run_id,
	  r.partition_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 12 HOUR) as is_12h,
	  r.partition_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 DAY) as is_1d,
	  r.partition_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 3 DAY) as is_3d,
	  r.partition_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY) as is_7d,
	  -- The identity of the first changelist that was tested, assuming the
	  -- result was part of a presubmit run, and the owner of the presubmit
	  -- run was a user and not automation.
	  IF(ARRAY_LENGTH(r.presubmit_run_cls)>0 AND r.presubmit_run_owner='user',
		  CONCAT(r.presubmit_run_cls[OFFSET(0)].host, r.presubmit_run_cls[OFFSET(0)].change),
		  NULL) as presubmit_run_user_cl_id,
	  (r.presubmit_run_id IS NOT NULL AND r.is_ingested_invocation_blocked) as is_presubmit_reject,
	  r.is_test_run_blocked as is_test_run_fail,
	FROM clustered_failures_latest
  )

  SELECT
	  cluster_algorithm,
	  cluster_id,

	  -- 1 day metrics.
	  COUNT(DISTINCT IF(is_1d AND is_presubmit_reject AND NOT is_exonerated, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_1d,
	  COUNT(DISTINCT IF(is_1d AND is_presubmit_reject AND NOT is_exonerated_pre_weetbix, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_pre_weetbix_1d,
	  COUNT(DISTINCT IF(is_1d AND is_presubmit_reject, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_pre_exon_1d,
	  COUNT(DISTINCT IF(is_1d AND is_presubmit_reject AND is_included_with_high_priority AND NOT is_exonerated, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_residual_1d,
	  COUNT(DISTINCT IF(is_1d AND is_presubmit_reject AND is_included_with_high_priority AND NOT is_exonerated_pre_weetbix, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_residual_pre_weetbix_1d,
	  COUNT(DISTINCT IF(is_1d AND is_presubmit_reject AND is_included_with_high_priority, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_residual_pre_exon_1d,
	  COUNT(DISTINCT IF(is_1d AND is_test_run_fail AND NOT is_exonerated, test_run_id, NULL)) as test_run_fails_1d,
	  COUNT(DISTINCT IF(is_1d AND is_test_run_fail AND NOT is_exonerated_pre_weetbix, test_run_id, NULL)) as test_run_fails_pre_weetbix_1d,
	  COUNT(DISTINCT IF(is_1d AND is_test_run_fail, test_run_id, NULL)) as  test_run_fails_pre_exon_1d,
	  COUNT(DISTINCT IF(is_1d AND is_test_run_fail AND is_included_with_high_priority AND NOT is_exonerated, test_run_id, NULL)) as test_run_fails_residual_1d,
	  COUNT(DISTINCT IF(is_1d AND is_test_run_fail AND is_included_with_high_priority AND NOT is_exonerated_pre_weetbix, test_run_id, NULL)) as test_run_fails_residual_pre_weetbix_1d,
	  COUNT(DISTINCT IF(is_1d AND is_test_run_fail AND is_included_with_high_priority, test_run_id, NULL)) as  test_run_fails_residual_pre_exon_1d,
	  COUNTIF(is_1d AND NOT is_exonerated) as failures_1d,
	  COUNTIF(is_1d AND NOT is_exonerated_pre_weetbix) as failures_pre_weetbix_1d,
	  COUNTIF(is_1d) AS failures_pre_exon_1d,
	  COUNTIF(is_1d AND is_included_with_high_priority AND NOT is_exonerated) as failures_residual_1d,
	  COUNTIF(is_1d AND is_included_with_high_priority AND NOT is_exonerated_pre_weetbix) as failures_residual_pre_weetbix_1d,
	  COUNTIF(is_1d AND is_included_with_high_priority) as failures_residual_pre_exon_1d,

	  -- 3 day metrics.
	  COUNT(DISTINCT IF(is_3d AND is_presubmit_reject AND NOT is_exonerated, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_3d,
	  COUNT(DISTINCT IF(is_3d AND is_presubmit_reject AND NOT is_exonerated_pre_weetbix, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_pre_weetbix_3d,
	  COUNT(DISTINCT IF(is_3d AND is_presubmit_reject, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_pre_exon_3d,
	  COUNT(DISTINCT IF(is_3d AND is_presubmit_reject AND is_included_with_high_priority AND NOT is_exonerated, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_residual_3d,
	  COUNT(DISTINCT IF(is_3d AND is_presubmit_reject AND is_included_with_high_priority AND NOT is_exonerated_pre_weetbix, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_residual_pre_weetbix_3d,
	  COUNT(DISTINCT IF(is_3d AND is_presubmit_reject AND is_included_with_high_priority, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_residual_pre_exon_3d,
	  COUNT(DISTINCT IF(is_3d AND is_test_run_fail AND NOT is_exonerated, test_run_id, NULL)) as test_run_fails_3d,
	  COUNT(DISTINCT IF(is_3d AND is_test_run_fail AND NOT is_exonerated_pre_weetbix, test_run_id, NULL)) as test_run_fails_pre_weetbix_3d,
	  COUNT(DISTINCT IF(is_3d AND is_test_run_fail, test_run_id, NULL)) as  test_run_fails_pre_exon_3d,
	  COUNT(DISTINCT IF(is_3d AND is_test_run_fail AND is_included_with_high_priority AND NOT is_exonerated, test_run_id, NULL)) as test_run_fails_residual_3d,
	  COUNT(DISTINCT IF(is_3d AND is_test_run_fail AND is_included_with_high_priority AND NOT is_exonerated_pre_weetbix, test_run_id, NULL)) as test_run_fails_residual_pre_weetbix_3d,
	  COUNT(DISTINCT IF(is_3d AND is_test_run_fail AND is_included_with_high_priority, test_run_id, NULL)) as  test_run_fails_residual_pre_exon_3d,
	  COUNTIF(is_3d AND NOT is_exonerated) as failures_3d,
	  COUNTIF(is_3d AND NOT is_exonerated_pre_weetbix) as failures_pre_weetbix_3d,
	  COUNTIF(is_3d) AS failures_pre_exon_3d,
	  COUNTIF(is_3d AND is_included_with_high_priority AND NOT is_exonerated) as failures_residual_3d,
	  COUNTIF(is_3d AND is_included_with_high_priority AND NOT is_exonerated_pre_weetbix) as failures_residual_pre_weetbix_3d,
	  COUNTIF(is_3d AND is_included_with_high_priority) as failures_residual_pre_exon_3d,

	  -- 7 day metrics.
	  COUNT(DISTINCT IF(is_7d AND is_presubmit_reject AND NOT is_exonerated, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_7d,
	  COUNT(DISTINCT IF(is_7d AND is_presubmit_reject AND NOT is_exonerated_pre_weetbix, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_pre_weetbix_7d,
	  COUNT(DISTINCT IF(is_7d AND is_presubmit_reject, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_pre_exon_7d,
	  COUNT(DISTINCT IF(is_7d AND is_presubmit_reject AND is_included_with_high_priority AND NOT is_exonerated, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_residual_7d,
	  COUNT(DISTINCT IF(is_7d AND is_presubmit_reject AND is_included_with_high_priority AND NOT is_exonerated_pre_weetbix, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_residual_pre_weetbix_7d,
	  COUNT(DISTINCT IF(is_7d AND is_presubmit_reject AND is_included_with_high_priority, presubmit_run_user_cl_id, NULL)) as presubmit_rejects_residual_pre_exon_7d,
	  COUNT(DISTINCT IF(is_7d AND is_test_run_fail AND NOT is_exonerated, test_run_id, NULL)) as test_run_fails_7d,
	  COUNT(DISTINCT IF(is_7d AND is_test_run_fail AND NOT is_exonerated_pre_weetbix, test_run_id, NULL)) as test_run_fails_pre_weetbix_7d,
	  COUNT(DISTINCT IF(is_7d AND is_test_run_fail, test_run_id, NULL)) as  test_run_fails_pre_exon_7d,
	  COUNT(DISTINCT IF(is_7d AND is_test_run_fail AND is_included_with_high_priority AND NOT is_exonerated, test_run_id, NULL)) as test_run_fails_residual_7d,
	  COUNT(DISTINCT IF(is_7d AND is_test_run_fail AND is_included_with_high_priority AND NOT is_exonerated_pre_weetbix, test_run_id, NULL)) as test_run_fails_residual_pre_weetbix_7d,
	  COUNT(DISTINCT IF(is_7d AND is_test_run_fail AND is_included_with_high_priority, test_run_id, NULL)) as  test_run_fails_residual_pre_exon_7d,
	  COUNTIF(is_7d AND NOT is_exonerated) as failures_7d,
	  COUNTIF(is_7d AND NOT is_exonerated_pre_weetbix) as failures_pre_weetbix_7d,
	  COUNTIF(is_7d) AS failures_pre_exon_7d,
	  COUNTIF(is_7d AND is_included_with_high_priority AND NOT is_exonerated) as failures_residual_7d,
	  COUNTIF(is_7d AND is_included_with_high_priority AND NOT is_exonerated_pre_weetbix) as failures_residual_pre_weetbix_7d,
	  COUNTIF(is_7d AND is_included_with_high_priority) as failures_residual_pre_exon_7d,

	  -- Other analysis.
	  ANY_VALUE(failure_reason) as example_failure_reason,
	  APPROX_TOP_COUNT(test_id, 5) as top_test_ids,
  FROM clustered_failures_extended
  WHERE is_included
  GROUP BY cluster_algorithm, cluster_id`
