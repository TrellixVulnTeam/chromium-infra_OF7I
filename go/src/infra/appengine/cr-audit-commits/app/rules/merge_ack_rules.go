// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"context"

	cpb "infra/appengine/cr-audit-commits/app/proto"
)

// AcknowledgeMerge is a Rule that acknowledges any merge into a release branch.
type AcknowledgeMerge struct {
	*cpb.AcknowledgeMerge
}

// GetName returns the name of the rule.
func (rule AcknowledgeMerge) GetName() string {
	return "AcknowledgeMerge"
}

// Run executes the rule.
func (rule AcknowledgeMerge) Run(ctx context.Context, ap *AuditParams, rc *RelevantCommit, cs *Clients) (*RuleResult, error) {
	result := &RuleResult{RuleResultStatus: RulePassed}
	return result, nil
}
