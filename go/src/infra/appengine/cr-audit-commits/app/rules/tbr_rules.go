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
	"go.chromium.org/luci/common/logging"

	cpb "infra/appengine/cr-audit-commits/app/proto"
)

const (
	// Do not ask gerrit about a change more than once every hour.
	pollInterval     = time.Hour
	chromeTBRMessage = `The commit was not approved by a reviewer other than the owner.

Beginning in Q1 2021, Chrome is disallowing TBRs. Learn more at go/chrome-cr-owners-site. Getting code review on all CLs will avoid having these bugs filed.

CHERRY-PICKS and REVERTS: Rubber Stamper is now available to approve clean cherry-picks and reverts if they were created via the Gerrit UI. Adding the bot (rubber-stamper@appspot.gserviceaccount.com) as a reviewer to your CL will cause it to scan and approve it. Just typing "Rubber St" will autocomplete the full address for you.

This bug is a warning that what you have done will stop working when Gerrit blocks such submissions; you can close it to signal that you've gotten review for the CL that this bug was filed against.`
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
	logging.Debugf(ctx, "Applying the ChangeReviewed rule to RelevantCommit: %+v", rc.RepoStateKey)
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
	logging.Debugf(ctx, "Gerrit change owner is: %d", owner)
	crLabelInfo, crExists := change.Labels["Code-Review"]
	botCommitLabelInfo, bcExists := change.Labels["Bot-Commit"]

	// Bypass code-review check if Bot-Commit label has max vote of 1
	if bcExists {
		logging.Debugf(ctx, "Bot-Commit label exists")
		bcMaxVal, err := getMaxLabelValue(botCommitLabelInfo.Values)
		if err != nil {
			return nil, err
		}
		logging.Debugf(ctx, "Bot-Commit max value is %d", bcMaxVal)

		for _, vote := range botCommitLabelInfo.All {
			logging.Debugf(ctx, "Bot-Commit label voter %d voted %d", vote.AccountID, vote.Value)
			if int(vote.Value) == bcMaxVal {
				logging.Debugf(ctx, "Bot-Commit label voter %d voted max value %d, rule passed", vote.AccountID, vote.Value)
				// Valid Bot-Commit found.
				result.RuleResultStatus = RulePassed
				return result, nil
			}
		}

		logging.Debugf(ctx, "Bot-Commit label exists with no votes that match max value")
	}

	if !crExists {
		return nil, fmt.Errorf("The gerrit change for Commit %v does not have the 'Code-Review' label", rc.CommitHash)
	}
	logging.Debugf(ctx, "Code-Review label exists")
	maxValue, err := getMaxLabelValue(crLabelInfo.Values)
	if err != nil {
		return nil, err
	}
	logging.Debugf(ctx, "Code-Review label max value is: %d", maxValue)
	for _, vote := range crLabelInfo.All {
		logging.Debugf(ctx, "Code-Review label voter %d voted %d", vote.AccountID, vote.Value)
		if int(vote.Value) == maxValue && vote.AccountID != owner {
			logging.Debugf(ctx, "Code-Review label voter %d voted %d who is not owner %d", vote.AccountID, vote.Value, owner)
			// Valid approver found.
			result.RuleResultStatus = RulePassed
			return result, nil
		}
	}
	result.RuleResultStatus = RuleFailed
	result.Message = r.Message
	if result.Message == "" {
		// TODO: Move to configuration.
		result.Message = chromeTBRMessage
	}

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
