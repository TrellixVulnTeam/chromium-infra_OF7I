// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"context"

	"go.chromium.org/luci/common/api/gerrit"

	cpb "infra/appengine/cr-audit-commits/app/proto"
)

const (
	// Do not ask gerrit about a change more than once every hour.
	pollInterval = time.Hour
)

// getMaxLabelValue determines the highest possible value of a vote for a given
// label. Gerrit represents the values as "-2", "-1", " 0", "+1", "+2" in the
// mapping of values to descriptions, yet the ApprovalInfo has it as an integer,
// hence the conversion.
func getMaxLabelValue(values map[string]string) (int, error) {
	maxIntVal := 0
	unset := true
	for k := range values {
		intVal, err := strconv.Atoi(strings.TrimSpace(k))
		if err != nil {
			return 0, err
		}
		if intVal > maxIntVal || unset {
			unset = false
			maxIntVal = intVal
		}
	}
	if unset {
		return 0, fmt.Errorf("expected at least one numerical value in the keys of %v", values)
	}
	return maxIntVal, nil

}

// ChangeReviewed is a RuleFunc that verifies that someone other than the
// owner has reviewed the change.
type ChangeReviewed struct {
	*cpb.ChangeReviewed
}

// GetName returns the name of the rule.
func (r ChangeReviewed) GetName() string {
	return "ChangeReviewed"
}

// shouldSkip decides if a given commit shouldn't be audited with this rule.
//
// E.g. if it's authored by an authorized automated account.
func (r ChangeReviewed) shouldSkip(rc *RelevantCommit) bool {
	for _, rob := range r.Robots {
		if rc.AuthorAccount == rob {
			return true
		}
	}
	return false
}

// Run executes the rule.
func (r ChangeReviewed) Run(ctx context.Context, ap *AuditParams, rc *RelevantCommit, cs *Clients) (*RuleResult, error) {
	result := &RuleResult{}
	result.RuleName = r.GetName()
	prevResult := PreviousResult(ctx, rc, result.RuleName)
	if prevResult != nil && (prevResult.RuleResultStatus != RulePending ||
		// If we checked gerrit recently, wait before checking again, leave the rule as pending.
		rc.LastExternalPoll.After(time.Now().Add(-pollInterval))) {
		return prevResult, nil
	} else if prevResult != nil {
		// Preserve any metadata from the previous execution of the rule.
		result.MetaData = prevResult.MetaData
	}
	if r.shouldSkip(rc) {
		result.RuleResultStatus = RuleSkipped
		return result, nil
	}
	rc.LastExternalPoll = time.Now()
	change, err := getChangeWithLabelDetails(ctx, ap, rc, cs)
	if err != nil {
		return nil, err
	}
	owner := change.Owner.AccountID
	crLabelInfo, crExists := change.Labels["Code-Review"]
	botCommitLabelInfo, bcExists := change.Labels["Bot-Commit"]

	// Bypass code-review check if Bot-Commit label has max vote of 1
	if bcExists {
		bcVal, err := getMaxLabelValue(botCommitLabelInfo.Values)
		if err != nil {
			return nil, err
		}
		if bcVal == 1 {
			result.RuleResultStatus = RulePassed
			return result, nil
		}
	}

	if !crExists {
		return nil, fmt.Errorf("The gerrit change for Commit %v does not have the 'Code-Review' label", rc.CommitHash)
	}
	maxValue, err := getMaxLabelValue(crLabelInfo.Values)
	if err != nil {
		return nil, err
	}
	for _, vote := range crLabelInfo.All {
		if int(vote.Value) == maxValue && vote.AccountID != owner {
			// Valid approver found.
			result.RuleResultStatus = RulePassed
			return result, nil
		}
	}
	result.RuleResultStatus = RuleFailed
	result.Message = "The commit was not approved by a reviewer other than the owner. Beginning in Q1 2020, Chrome is disallowing TBRs. Learn more at go/chrome-cr-owners-site."
	return result, nil
}

func getChangeWithLabelDetails(ctx context.Context, ap *AuditParams, rc *RelevantCommit, cs *Clients) (*gerrit.Change, error) {
	cls, _, err := cs.gerrit.ChangeQuery(ctx, gerrit.ChangeQueryParams{
		Query: fmt.Sprintf("commit:%s", rc.CommitHash),
		Options: []string{
			"DETAILED_LABELS",
		},
	})
	if err != nil {
		return nil, err
	}
	if len(cls) == 0 {
		return nil, errors.New("no CL found for commit")
	}
	return cls[0], nil
}
