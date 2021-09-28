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

func validateConfig(ctx *validation.Context, cfg *Config) {
	validateMonorailHostname(ctx, cfg.MonorailHostname)
}

func validateMonorailHostname(ctx *validation.Context, hostname string) {
	ctx.Enter("monorail_hostname")
	if hostname == "" {
		ctx.Errorf("empty value is not allowed")
	} else if _, err := url.Parse(hostname); err != nil {
		ctx.Errorf("invalid hostname: %s", hostname)
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
	validateProjectConfig(ctx, msg)
	return msg
}

func validateProjectConfig(ctx *validation.Context, cfg *ProjectConfig) {
	validateMonorail(ctx, cfg.Monorail)
}

func validateMonorail(ctx *validation.Context, cfg *MonorailProject) {
	ctx.Enter("monorail")
	defer ctx.Exit()

	if cfg == nil {
		ctx.Errorf("monorail must be specified")
		return
	}

	validateMonorailProject(ctx, cfg.GetProject())
	validateDefaultFieldValues(ctx, cfg.GetDefaultFieldValues())
	validateFieldID(ctx, cfg.GetPriorityFieldId(), "priority_field_id")
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
