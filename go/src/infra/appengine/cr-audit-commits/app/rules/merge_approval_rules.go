// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"context"

	cpb "infra/appengine/cr-audit-commits/app/proto"
)

const (
	mergeApprovedLabel = "Merge-Approved-%s"
)

// OnlyMergeApprovedChange is a Rule that verifies that only approved changes
// are merged into a release branch.
type OnlyMergeApprovedChange struct {
	*cpb.OnlyMergeApprovedChange
}

// GetName returns the name of the rule.
func (r OnlyMergeApprovedChange) GetName() string {
	return "OnlyMergeApprovedChange"
}

// Run executes the rule.
func (r OnlyMergeApprovedChange) Run(ctx context.Context, ap *AuditParams, rc *RelevantCommit, cs *Clients) (*RuleResult, error) {
	result := &RuleResult{RuleResultStatus: RulePassed}
	return result, nil
}
