// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"infra/appengine/weetbix/internal/config"
	mpb "infra/monorailv2/api/v3/api_proto"
)

// ChromiumTestPriorityField is the resource name of the priority field
// that is consistent with ChromiumTestConfig.
const ChromiumTestPriorityField = "projects/chromium/fieldDefs/11"

// ChromiumTestTypeField is the resource name of the type field
// that is consistent with ChromiumTestConfig.
const ChromiumTestTypeField = "projects/chromium/fieldDefs/10"

// ChromiumTestConfig provides chromium-like configuration for tests
// to use.
func ChromiumTestConfig() map[string]*config.MonorailProject {
	projectCfg := map[string]*config.MonorailProject{
		"chromium": {
			Project: "chromium",
			DefaultFieldValues: []*config.MonorailFieldValue{
				{
					FieldId: 10,
					Value:   "Bug",
				},
			},
			PriorityFieldId: 11,
		},
	}
	return projectCfg
}

// ChromiumTestIssuePriority returns the priority of an issue, assuming
// it has been created consistent with ChromiumTestConfig.
func ChromiumTestIssuePriority(issue *mpb.Issue) string {
	for _, fv := range issue.FieldValues {
		if fv.Field == ChromiumTestPriorityField {
			return fv.Value
		}
	}
	return ""
}
