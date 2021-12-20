// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"fmt"
	"regexp"

	"google.golang.org/protobuf/types/known/durationpb"

	luciproto "go.chromium.org/luci/common/proto"
	"go.chromium.org/luci/config/validation"

	"infra/appengine/weetbix/pbutil"
)

const maxHysteresisPercent = 1000

var (
	// https://cloud.google.com/storage/docs/naming-buckets
	bucketRE = regexp.MustCompile(`^[a-z0-9][a-z0-9\-_.]{1,220}[a-z0-9]$`)

	// From https://source.chromium.org/chromium/infra/infra/+/main:appengine/monorail/project/project_constants.py;l=13.
	monorailProjectRE = regexp.MustCompile(`^[a-z0-9][-a-z0-9]{0,61}[a-z0-9]$`)

	// https://source.chromium.org/chromium/infra/infra/+/main:luci/appengine/auth_service/proto/realms_config.proto;l=85;drc=04e290f764a293d642d287b0118e9880df4afb35
	realmRE = regexp.MustCompile(`^[a-z0-9_\.\-/]{1,400}$`)

	// Matches valid prefixes to use when displaying bugs.
	// E.g. "crbug.com", "fxbug.dev".
	prefixRE = regexp.MustCompile(`^[a-z0-9\-.]{0,64}$`)

	// hostnameRE excludes most invalid hostnames.
	hostnameRE = regexp.MustCompile(`^[a-z][a-z9-9\-.]{0,62}[a-z]$`)

	// Patterns for BigQuery table.
	// https://cloud.google.com/resource-manager/docs/creating-managing-projects
	cloudProjectRE = regexp.MustCompile(`^[a-z][a-z0-9\-]{4,28}[a-z0-9]$`)
	// https://cloud.google.com/bigquery/docs/datasets#dataset-naming
	datasetRE = regexp.MustCompile(`^[a-zA-Z0-9_]*$`)
	// https://cloud.google.com/bigquery/docs/tables#table_naming
	tableRE = regexp.MustCompile(`^[\p{L}\p{M}\p{N}\p{Pc}\p{Pd}\p{Zs}]*$`)
)

func validateConfig(ctx *validation.Context, cfg *Config) {
	validateHostname(ctx, "monorail_hostname", cfg.MonorailHostname, false /*optional*/)
	validateStringConfig(ctx, "chunk_gcs_bucket", cfg.ChunkGcsBucket, bucketRE)
	// Limit to default max_concurrent_requests of 1000.
	// https://cloud.google.com/appengine/docs/standard/go111/config/queueref
	validateIntegerConfig(ctx, "reclustering_workers", cfg.ReclusteringWorkers, 1000)
	// Limit within GAE autoscaling request timeout of 10 minutes.
	// https://cloud.google.com/appengine/docs/standard/python/how-instances-are-managed
	validateIntegerConfig(ctx, "reclustering_interval_minutes", cfg.ReclusteringIntervalMinutes, 9)
}

func validateHostname(ctx *validation.Context, name, hostname string, optional bool) {
	ctx.Enter(name)
	if hostname == "" {
		if !optional {
			ctx.Errorf("empty value is not allowed")
		}
	} else if !hostnameRE.MatchString(hostname) {
		ctx.Errorf("invalid hostname: %q", hostname)
	}
	ctx.Exit()
}

func validateStringConfig(ctx *validation.Context, name, cfg string, re *regexp.Regexp) {
	ctx.Enter(name)
	switch err := pbutil.ValidateWithRe(re, cfg); err {
	case pbutil.Unspecified:
		ctx.Errorf("empty %s is not allowed", name)
	case pbutil.DoesNotMatch:
		ctx.Errorf("invalid %s: %q", name, cfg)
	}
	ctx.Exit()
}

func validateIntegerConfig(ctx *validation.Context, name string, cfg, max int64) {
	ctx.Enter(name)
	defer ctx.Exit()

	if cfg < 0 {
		ctx.Errorf("value is less than zero")
	}
	if cfg >= max {
		ctx.Errorf("value is greater than %v", max)
	}
}

func validateDuration(ctx *validation.Context, name string, du *durationpb.Duration) {
	ctx.Enter(name)
	defer ctx.Exit()

	switch {
	case du == nil:
		ctx.Errorf("empty %s is not allowed", name)
	case du.CheckValid() != nil:
		ctx.Errorf("%s is invalid", name)
	case du.AsDuration() < 0:
		ctx.Errorf("%s is less than 0", name)
	}
}

func validateUpdateTestVariantTask(ctx *validation.Context, utCfg *UpdateTestVariantTask) {
	ctx.Enter("update_test_variant")
	defer ctx.Exit()
	if utCfg == nil {
		return
	}
	validateDuration(ctx, "interval", utCfg.UpdateTestVariantTaskInterval)
	validateDuration(ctx, "duration", utCfg.TestVariantStatusUpdateDuration)
}

func validateBigQueryTable(ctx *validation.Context, tCfg *BigQueryExport_BigQueryTable) {
	ctx.Enter("table")
	defer ctx.Exit()
	if tCfg == nil {
		ctx.Errorf("empty bigquery table is not allowed")
		return
	}
	validateStringConfig(ctx, "cloud_project", tCfg.CloudProject, cloudProjectRE)
	validateStringConfig(ctx, "dataset", tCfg.Dataset, datasetRE)
	validateStringConfig(ctx, "table_name", tCfg.Table, tableRE)
}

func validateBigQueryExport(ctx *validation.Context, bqCfg *BigQueryExport) {
	ctx.Enter("bigquery_export")
	defer ctx.Exit()
	if bqCfg == nil {
		return
	}
	validateBigQueryTable(ctx, bqCfg.Table)
	if bqCfg.GetPredicate() == nil {
		return
	}
	if err := pbutil.ValidateAnalyzedTestVariantPredicate(bqCfg.Predicate); err != nil {
		ctx.Errorf(fmt.Sprintf("%s", err))
	}
}

func validateTestVariantAnalysisConfig(ctx *validation.Context, tvCfg *TestVariantAnalysisConfig) {
	ctx.Enter("test_variant")
	defer ctx.Exit()
	if tvCfg == nil {
		return
	}
	validateUpdateTestVariantTask(ctx, tvCfg.UpdateTestVariantTask)
	for _, bqe := range tvCfg.BqExports {
		validateBigQueryExport(ctx, bqe)
	}
}

func validateRealmConfig(ctx *validation.Context, rCfg *RealmConfig) {
	ctx.Enter(fmt.Sprintf("realm %s", rCfg.Name))
	defer ctx.Exit()

	validateStringConfig(ctx, "realm_name", rCfg.Name, realmRE)
	validateTestVariantAnalysisConfig(ctx, rCfg.TestVariantAnalysis)
}

// validateProjectConfigRaw deserializes the project-level config message
// and passes it through the validator.
func validateProjectConfigRaw(ctx *validation.Context, content string) *ProjectConfig {
	msg := &ProjectConfig{}
	if err := luciproto.UnmarshalTextML(content, msg); err != nil {
		ctx.Errorf("failed to unmarshal as text proto: %s", err)
		return nil
	}
	ValidateProjectConfig(ctx, msg)
	return msg
}

func ValidateProjectConfig(ctx *validation.Context, cfg *ProjectConfig) {
	validateMonorail(ctx, cfg.Monorail, cfg.BugFilingThreshold)
	validateImpactThreshold(ctx, cfg.BugFilingThreshold, "bug_filing_threshold")
	for _, rCfg := range cfg.Realms {
		validateRealmConfig(ctx, rCfg)
	}
}

func validateMonorail(ctx *validation.Context, cfg *MonorailProject, bugFilingThres *ImpactThreshold) {
	ctx.Enter("monorail")
	defer ctx.Exit()

	if cfg == nil {
		ctx.Errorf("monorail must be specified")
		return
	}

	validateStringConfig(ctx, "project", cfg.Project, monorailProjectRE)
	validateDefaultFieldValues(ctx, cfg.DefaultFieldValues)
	validateFieldID(ctx, cfg.PriorityFieldId, "priority_field_id")
	validatePriorities(ctx, cfg.Priorities, bugFilingThres)
	validatePriorityHysteresisPercent(ctx, cfg.PriorityHysteresisPercent)
	validateDisplayPrefix(ctx, cfg.DisplayPrefix)
	validateHostname(ctx, "monorail_hostname", cfg.MonorailHostname, true /*optional*/)
}

func validateDefaultFieldValues(ctx *validation.Context, fvs []*MonorailFieldValue) {
	ctx.Enter("default_field_values")
	for i, fv := range fvs {
		ctx.Enter("[%v]", i)
		validateFieldValue(ctx, fv)
		ctx.Exit()
	}
	ctx.Exit()
}

func validateFieldID(ctx *validation.Context, fieldID int64, fieldName string) {
	ctx.Enter(fieldName)
	if fieldID < 0 {
		ctx.Errorf("value must be non-negative")
	}
	ctx.Exit()
}

func validateFieldValue(ctx *validation.Context, fv *MonorailFieldValue) {
	validateFieldID(ctx, fv.GetFieldId(), "field_id")
	// No validation applies to field value.
}

func validatePriorities(ctx *validation.Context, ps []*MonorailPriority, bugFilingThres *ImpactThreshold) {
	ctx.Enter("priorities")
	if len(ps) == 0 {
		ctx.Errorf("at least one monorail priority must be specified")
	}
	for i, p := range ps {
		ctx.Enter("[%v]", i)
		validatePriority(ctx, p)
		if i == len(ps)-1 {
			// The lowest priority threshold must be satisfied by
			// the bug-filing threshold. This ensures that bugs meeting the
			// bug-filing threshold meet the bug keep-open threshold.
			validatePrioritySatisfiedByBugFilingThreshold(ctx, p, bugFilingThres)
		}
		ctx.Exit()
	}
	ctx.Exit()
}

func validatePriority(ctx *validation.Context, p *MonorailPriority) {
	validatePriorityValue(ctx, p.Priority)
	validateImpactThreshold(ctx, p.Threshold, "threshold")
}

func validatePrioritySatisfiedByBugFilingThreshold(ctx *validation.Context, p *MonorailPriority, bugFilingThres *ImpactThreshold) {
	ctx.Enter("threshold")
	defer ctx.Exit()
	t := p.Threshold
	if t == nil || bugFilingThres == nil {
		// Priority without threshold and no bug filing threshold specified
		// are already reported as errors elsewhere.
		return
	}
	validateBugFilingThresholdSatisfiesMetricThresold(ctx, t.TestResultsFailed, bugFilingThres.TestResultsFailed, "test_results_failed")
	validateBugFilingThresholdSatisfiesMetricThresold(ctx, t.TestRunsFailed, bugFilingThres.TestRunsFailed, "test_runs_failed")
	validateBugFilingThresholdSatisfiesMetricThresold(ctx, t.PresubmitRunsFailed, bugFilingThres.PresubmitRunsFailed, "presubmit_runs_failed")
}

func validatePriorityValue(ctx *validation.Context, value string) {
	ctx.Enter("priority")
	// Although it is possible to allow the priority field to be empty, it
	// would be rather unusual for a project to set itself up this way. For
	// now, prefer to enforce priority values are non-empty as this will pick
	// likely configuration errors.
	if value == "" {
		ctx.Errorf("empty value is not allowed")
	}
	ctx.Exit()
}

func validateImpactThreshold(ctx *validation.Context, t *ImpactThreshold, fieldName string) {
	ctx.Enter(fieldName)
	defer ctx.Exit()

	if t == nil {
		ctx.Errorf("impact thresolds must be specified")
		return
	}

	validateMetricThreshold(ctx, t.TestResultsFailed, "test_results_failed")
	validateMetricThreshold(ctx, t.TestRunsFailed, "test_runs_failed")
	validateMetricThreshold(ctx, t.PresubmitRunsFailed, "presubmit_runs_failed")
}

func validateMetricThreshold(ctx *validation.Context, t *MetricThreshold, fieldName string) {
	ctx.Enter(fieldName)
	defer ctx.Exit()

	if t == nil {
		// Not specified.
		return
	}

	validateNonNegative(ctx, t.OneDay, "one_day")
	validateNonNegative(ctx, t.ThreeDay, "three_day")
	validateNonNegative(ctx, t.SevenDay, "seven_day")
}

func validatePriorityHysteresisPercent(ctx *validation.Context, value int64) {
	ctx.Enter("priority_hysteresis_percent")
	if value > maxHysteresisPercent {
		ctx.Errorf("value must not exceed %v percent", maxHysteresisPercent)
	}
	if value < 0 {
		ctx.Errorf("value must not be negative")
	}
	ctx.Exit()
}

func validateNonNegative(ctx *validation.Context, value *int64, fieldName string) {
	ctx.Enter(fieldName)
	if value != nil && *value < 0 {
		ctx.Errorf("value must be non-negative")
	}
	ctx.Exit()
}

func validateBugFilingThresholdSatisfiesMetricThresold(ctx *validation.Context, threshold *MetricThreshold, bugFilingThres *MetricThreshold, fieldName string) {
	ctx.Enter(fieldName)
	defer ctx.Exit()
	if threshold == nil {
		threshold = &MetricThreshold{}
	}
	if bugFilingThres == nil {
		// Bugs are not filed based on this metric. So
		// we do not need to check that bugs filed
		// based on this metric will stay open.
		return
	}
	validateBugFilingThresholdSatisfiesThresold(ctx, threshold.OneDay, bugFilingThres.OneDay, "one_day")
	validateBugFilingThresholdSatisfiesThresold(ctx, threshold.ThreeDay, bugFilingThres.ThreeDay, "three_day")
	validateBugFilingThresholdSatisfiesThresold(ctx, threshold.SevenDay, bugFilingThres.SevenDay, "seven_day")
}

func validateBugFilingThresholdSatisfiesThresold(ctx *validation.Context, threshold *int64, bugFilingThres *int64, fieldName string) {
	ctx.Enter(fieldName)
	defer ctx.Exit()
	if bugFilingThres == nil {
		// Bugs are not filed based on this threshold.
		return
	}
	if *bugFilingThres < 0 {
		// The bug-filing threshold is invalid. This is already reported as an
		// error elsewhere.
		return
	}
	// If a bug may be filed at a particular threshold, it must also be
	// allowed to stay open at that threshold.
	if threshold == nil {
		ctx.Errorf("%s threshold must be set, with a value of at most %v (the configured bug-filing threshold). This ensures that bugs which are filed meet the criteria to stay open", fieldName, *bugFilingThres)
	} else if *threshold > *bugFilingThres {
		ctx.Errorf("value must be at most %v (the configured bug-filing threshold). This ensures that bugs which are filed meet the criteria to stay open", *bugFilingThres)
	}
}

func validateDisplayPrefix(ctx *validation.Context, prefix string) {
	ctx.Enter(prefix)
	defer ctx.Exit()
	if !prefixRE.MatchString(prefix) {
		ctx.Errorf("invalid display prefix: %q", prefix)
	}
}
