// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"net/url"
	"regexp"

	protov1 "github.com/golang/protobuf/proto"
	luciproto "go.chromium.org/luci/common/proto"
	"go.chromium.org/luci/config/validation"
)

// From https://source.chromium.org/chromium/infra/infra/+/main:appengine/monorail/project/project_constants.py;l=13.
var monorailProjectRE = regexp.MustCompile(`^[a-z0-9][-a-z0-9]{0,61}[a-z0-9]$`)

// https://cloud.google.com/storage/docs/naming-buckets
var bucketRE = regexp.MustCompile(`^[a-z0-9][a-z0-9\-_.]{1,220}[a-z0-9]$`)

const maxHysteresisPercent = 1000

func validateConfig(ctx *validation.Context, cfg *Config) {
	validateMonorailHostname(ctx, cfg.MonorailHostname)
	validateChunkGCSBucket(ctx, cfg.ChunkGcsBucket)
}

func validateMonorailHostname(ctx *validation.Context, hostname string) {
	ctx.Enter("monorail_hostname")
	if hostname == "" {
		ctx.Errorf("empty value is not allowed")
	} else if _, err := url.Parse("https://" + hostname + "/"); err != nil {
		ctx.Errorf("invalid hostname: %q", hostname)
	}
	ctx.Exit()
}

func validateChunkGCSBucket(ctx *validation.Context, bucket string) {
	ctx.Enter("chunk_gcs_bucket")
	if bucket == "" {
		ctx.Errorf("empty value is not allowed")
	} else if !bucketRE.MatchString(bucket) {
		ctx.Errorf("invalid bucket: %q", bucket)
	}
	ctx.Exit()
}

// validateProjectConfigRaw deserializes the project-level config message
// and passes it through the validator.
func validateProjectConfigRaw(ctx *validation.Context, content string) *ProjectConfig {
	msg := &ProjectConfig{}
	if err := luciproto.UnmarshalTextML(content, protov1.MessageV1(msg)); err != nil {
		ctx.Errorf("failed to unmarshal as text proto: %s", err)
		return nil
	}
	ValidateProjectConfig(ctx, msg)
	return msg
}

func ValidateProjectConfig(ctx *validation.Context, cfg *ProjectConfig) {
	validateMonorail(ctx, cfg.Monorail, cfg.BugFilingThreshold)
	validateImpactThreshold(ctx, cfg.BugFilingThreshold, "bug_filing_threshold")
}

func validateMonorail(ctx *validation.Context, cfg *MonorailProject, bugFilingThres *ImpactThreshold) {
	ctx.Enter("monorail")
	defer ctx.Exit()

	if cfg == nil {
		ctx.Errorf("monorail must be specified")
		return
	}

	validateMonorailProject(ctx, cfg.Project)
	validateDefaultFieldValues(ctx, cfg.DefaultFieldValues)
	validateFieldID(ctx, cfg.PriorityFieldId, "priority_field_id")
	validatePriorities(ctx, cfg.Priorities, bugFilingThres)
	validatePriorityHysteresisPercent(ctx, cfg.PriorityHysteresisPercent)
}

func validateMonorailProject(ctx *validation.Context, project string) {
	ctx.Enter("project")
	if project == "" {
		ctx.Errorf("empty value is not allowed")
	} else if !monorailProjectRE.MatchString(project) {
		ctx.Errorf("project is not a valid monorail project")
	}
	ctx.Exit()
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
	validateBugFilingThresholdSatisfiesFailureCountThresold(ctx, t.UnexpectedFailures_1D, bugFilingThres.UnexpectedFailures_1D, "unexpected_failures_1d")
	validateBugFilingThresholdSatisfiesFailureCountThresold(ctx, t.UnexpectedFailures_3D, bugFilingThres.UnexpectedFailures_3D, "unexpected_failures_3d")
	validateBugFilingThresholdSatisfiesFailureCountThresold(ctx, t.UnexpectedFailures_7D, bugFilingThres.UnexpectedFailures_7D, "unexpected_failures_7d")
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

	validateFailureCountThresold(ctx, t.UnexpectedFailures_1D, "unexpected_failures_1d")
	validateFailureCountThresold(ctx, t.UnexpectedFailures_3D, "unexpected_failures_3d")
	validateFailureCountThresold(ctx, t.UnexpectedFailures_7D, "unexpected_failures_7d")
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

func validateFailureCountThresold(ctx *validation.Context, threshold *int64, fieldName string) {
	ctx.Enter(fieldName)
	if threshold != nil && *threshold < 0 {
		ctx.Errorf("value must be non-negative")
	}
	ctx.Exit()
}

func validateBugFilingThresholdSatisfiesFailureCountThresold(ctx *validation.Context, threshold *int64, bugFilingThres *int64, fieldName string) {
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
