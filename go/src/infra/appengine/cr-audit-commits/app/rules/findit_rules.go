// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"context"
	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/golang/protobuf/ptypes"
	ds "go.chromium.org/gae/service/datastore"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/api/gerrit"
	"go.chromium.org/luci/common/api/gitiles"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
)

// Role is an enum describing the relationship between an email account and a
// commit. (Such as Committer or Author)
type Role uint8

const (
	// Committer is when the account is present in the committer field of
	// the commit.
	Committer Role = iota

	// Author is when the account is present in the author field of the
	// commit.
	Author
)

const (
	// MaxAutoCommitsPerDay indicates how many commits may be landed by the
	// findit service account in any 24 hour period.
	MaxAutoCommitsPerDay = 8
	// MaxAutoRevertsPerDay indicates how many reverts may be created by the
	// findit service account in any 24 hour period.
	MaxAutoRevertsPerDay = 20

	// MaxCulpritAge indicates the maximum delay allowed between a culprit
	// and findit reverting it.
	MaxCulpritAge = 24 * time.Hour

	// MaxRetriesPerCommit indicates how many times the auditor can retry
	// audting a commit if some rules return errors. This retry is meant to
	// handle transient errors on the underlying services.
	MaxRetriesPerCommit = 6 // Thirty minutes if checking every 5 minutes.
)

var (
	// This is the numeric result code for FAILURE. (As buildbot defines it)
	failedResultCode = 2
	stepsFieldMask   = field_mask.FieldMask{Paths: []string{"steps"}}
)

// countRelevantCommits follows the relevant commits previous pointer until a
// commit older than the cutoff time is found, and counts those that match the
// account and action given as parameters.
//
// Return an error when there is a datastore error.
func countRelevantCommits(ctx context.Context, rc *RelevantCommit, cutoff time.Time, account string, role Role) (int, error) {
	counter := 0
	current := rc
	for {
		switch {
		case current.CommitTime.Before(cutoff):
			return counter, nil
		case role == Committer:
			if current.CommitterAccount == account {
				counter++
			}
		case role == Author:
			if current.AuthorAccount == account {
				counter++
			}
		}

		if current.PreviousRelevantCommit == "" {
			return counter, nil
		}

		current = &RelevantCommit{
			CommitHash:   current.PreviousRelevantCommit,
			RepoStateKey: rc.RepoStateKey,
		}
		err := ds.Get(ctx, current)
		if err != nil {
			return 0, fmt.Errorf("Could not retrieve relevant commit with hash %s",
				current.CommitHash)
		}

	}
}

func countCommittedBy(ctx context.Context, rc *RelevantCommit, cutoff time.Time, account string) (int, error) {
	return countRelevantCommits(ctx, rc, cutoff, account, Committer)
}

func countAuthoredBy(ctx context.Context, rc *RelevantCommit, cutoff time.Time, account string) (int, error) {
	return countRelevantCommits(ctx, rc, cutoff, account, Author)
}

// AutoCommitsPerDay is a Rule that verifies that at most
// MaxAutoCommitsPerDay commits in the 24 hours preceding the triggering commit
// were committed by the triggering account.
type AutoCommitsPerDay struct{}

// GetName returns the name of the rule
func (rule AutoCommitsPerDay) GetName() string {
	return "AutoCommitsPerDay"
}

// Run executes the rule.
func (rule AutoCommitsPerDay) Run(ctx context.Context, ap *AuditParams, rc *RelevantCommit, cs *Clients) (*RuleResult, error) {
	result := &RuleResult{}
	cutoff := rc.CommitTime.Add(time.Duration(-24) * time.Hour)
	autoCommits, err := countCommittedBy(ctx, rc, cutoff, ap.TriggeringAccount)
	if err != nil {
		return nil, err
	}
	if autoCommits > MaxAutoCommitsPerDay {
		result.RuleResultStatus = RuleFailed
		result.Message = fmt.Sprintf(
			"%d commits were committed by account %s in 24 hours, and the maximum allowed is %d",
			autoCommits, ap.TriggeringAccount, MaxAutoCommitsPerDay)
	} else {
		result.RuleResultStatus = RulePassed
	}
	return result, nil
}

// AutoRevertsPerDay is a Rule that verifies that at most
// MaxAutoRevertsPerDay commits in the 24 hours preceding the triggering commit
// were authored by the triggering account.
type AutoRevertsPerDay struct{}

// GetName returns the name of the rule
func (rule AutoRevertsPerDay) GetName() string {
	return "AutoRevertsPerDay"
}

// Run executes the rule.
func (rule AutoRevertsPerDay) Run(ctx context.Context, ap *AuditParams, rc *RelevantCommit, cs *Clients) (*RuleResult, error) {
	result := &RuleResult{}
	cutoff := rc.CommitTime.Add(time.Duration(-24) * time.Hour)
	autoReverts, err := countAuthoredBy(ctx, rc, cutoff, ap.TriggeringAccount)
	if err != nil {
		return nil, err
	}
	if autoReverts > MaxAutoRevertsPerDay {
		result.RuleResultStatus = RuleFailed
		result.Message = fmt.Sprintf(
			"%d commits were created by %s account in 24 hours, and the maximum allowed is %d",
			autoReverts, ap.TriggeringAccount, MaxAutoRevertsPerDay)
	} else {
		result.RuleResultStatus = RulePassed
	}
	return result, nil
}

// CulpritAge is a Rule that verifies that the culprit being reverted is
// less than 24 hours older than the revert.
type CulpritAge struct{}

// GetName returns the name of the rule
func (rule CulpritAge) GetName() string {
	return "CulpritAge"
}

// Run executes the rule.
func (rule CulpritAge) Run(ctx context.Context, ap *AuditParams, rc *RelevantCommit, cs *Clients) (*RuleResult, error) {
	result := &RuleResult{}

	_, culprit, err := getRevertAndCulpritChanges(ctx, ap, rc, cs)
	if err != nil {
		return nil, err
	}
	if culprit == nil {
		return nil, fmt.Errorf("Commit %q does not appear to be a revert according to gerrit",
			rc.CommitHash)
	}

	host, project, err := gitiles.ParseRepoURL(ap.RepoCfg.BaseRepoURL)
	if err != nil {
		return nil, fmt.Errorf("The repo url %s somehow became invalid", ap.RepoCfg.BaseRepoURL)
	}

	gc, err := cs.NewGitilesClient(host)
	if err != nil {
		return nil, err
	}
	resp, err := gc.Log(ctx, &gitilespb.LogRequest{
		Project:    project,
		Committish: culprit.CurrentRevision,
		PageSize:   1,
	})
	if err != nil {
		return nil, err
	}
	c := resp.Log
	if len(c) == 0 {
		return nil, fmt.Errorf("commit %s not found in repo", culprit.CurrentRevision)
	}
	commitTime, err := ptypes.Timestamp(c[0].Committer.Time)
	if err != nil {
		return nil, err
	}
	if rc.CommitTime.Sub(commitTime) > MaxCulpritAge {
		result.RuleResultStatus = RuleFailed
		result.Message = fmt.Sprintf("The revert %s landed more than %s after the culprit %s landed",
			rc.CommitHash, MaxCulpritAge, c[0].Id)

	} else {
		result.RuleResultStatus = RulePassed
	}
	return result, nil
}

// CulpritInBuild is a Rule that verifies that the culprit is included in
// the list of changes of the failed build.
type CulpritInBuild struct{}

// GetName returns the name of the rule
func (rule CulpritInBuild) GetName() string {
	return "CulpritInBuild"
}

// Run executes the rule.
func (rule CulpritInBuild) Run(ctx context.Context, ap *AuditParams, rc *RelevantCommit, cs *Clients) (*RuleResult, error) {
	result := &RuleResult{}

	if isFlakeRevert(rc.CommitMessage) {
		// Bypass this rule for reverts of culprits of flake failures.
		result.RuleResultStatus = RuleSkipped
		return result, nil
	}

	_, culprit, err := getRevertAndCulpritChanges(ctx, ap, rc, cs)
	if err != nil {
		return nil, err
	}
	if culprit == nil {
		return nil, fmt.Errorf("Commit %q does not appear to be a revert according to gerrit",
			rc.CommitHash)
	}

	buildURL, err := failedBuildFromCommitMessage(rc.CommitMessage)
	changes, err := getBlamelist(ctx, buildURL, cs)
	if err != nil {
		return nil, err
	}

	changeFound := false
	for _, c := range changes {
		if c == culprit.CurrentRevision {
			changeFound = true
			break
		}
	}
	if changeFound {
		result.RuleResultStatus = RulePassed
	} else {
		result.RuleResultStatus = RuleFailed
		if buildURL != "" {
			result.Message = fmt.Sprintf("Hash %s not found in changes for build %q",
				culprit.CurrentRevision, buildURL)
		} else {
			result.Message = fmt.Sprintf(
				"The revert does not point to a failed build, expected link prefixed with \"%s\"",
				FailedBuildPrefix)
		}
	}
	return result, nil
}

// getRevertAndCulpritChanges gets (through Gerrit) the details of the revert
// CL and  the CL it reverts.
//
// Note: The RevertOf property of a Change does not guarantee that the cl is a
// pure revert of another; instead, the get-pure-revert api of Gerrit needs to
// be checked, like RevertOfCulprit below does.
func getRevertAndCulpritChanges(ctx context.Context, ap *AuditParams, rc *RelevantCommit, cs *Clients) (*gerrit.Change, *gerrit.Change, error) {
	cls, _, err := cs.gerrit.ChangeQuery(ctx, gerrit.ChangeQueryParams{Query: fmt.Sprintf("commit:%s", rc.CommitHash)})
	if err != nil {
		return nil, nil, err
	}
	if len(cls) == 0 {
		return nil, nil, fmt.Errorf("no CL found for commit %q", rc.CommitHash)
	}
	revert, err := cs.gerrit.ChangeDetails(ctx, cls[0].ChangeID, gerrit.ChangeDetailsParams{})

	if err != nil {
		return nil, nil, err
	}
	if revert.RevertOf == 0 {
		return revert, nil, nil
	}

	culprit, err := cs.gerrit.ChangeDetails(ctx, strconv.Itoa(revert.RevertOf),
		gerrit.ChangeDetailsParams{Options: []string{"CURRENT_REVISION"}})
	if err != nil {
		return nil, nil, err
	}
	if culprit.CurrentRevision == "" {
		return nil, nil, fmt.Errorf("Could not get current_revision property for cl %q",
			culprit.ChangeNumber)
	}
	return revert, culprit, nil
}

// FailedBuildIsAppropriateFailure is a Rule that verifies that the referred
// build contains a failed step appropriately named.
type FailedBuildIsAppropriateFailure struct{}

// GetName returns the name of the rule
func (rule FailedBuildIsAppropriateFailure) GetName() string {
	return "FailedBuildIsAppropriateFailure"
}

// Run executes the rule.
func (rule FailedBuildIsAppropriateFailure) Run(ctx context.Context, ap *AuditParams, rc *RelevantCommit, cs *Clients) (*RuleResult, error) {
	result := &RuleResult{}
	failableStepName := getFailedSteps(rc.CommitMessage)
	buildURL, err := failedBuildFromCommitMessage(rc.CommitMessage)
	if err != nil || buildURL == "" {
		result.RuleResultStatus = RuleFailed
		result.Message = fmt.Sprintf(
			"The revert does not point to a failed build, expected link prefixed with \"%s\"", FailedBuildPrefix)
		return result, nil
	}

	build, err := getBuildByURL(ctx, buildURL, cs, &stepsFieldMask)
	if err != nil {
		return nil, err
	}

	for _, s := range build.Steps {
		// Nested steps are named [<ancestor>|]*<child>
		stepPath := strings.Split(s.Name, "|")
		lastPart := stepPath[len(stepPath)-1]
		if lastPart == failableStepName || s.Name == failableStepName {
			if s.Status == buildbucketpb.Status_FAILURE {
				result.RuleResultStatus = RulePassed
				return result, nil
			}
		}
	}

	result.RuleResultStatus = RuleFailed
	result.Message = fmt.Sprintf("Referred build %q does not have an expected failure in the following step: %s",
		buildURL, failableStepName)
	return result, nil
}

// RevertOfCulprit is a Rule that verifies that the reverting commit is a
// revert of the named culprit.
type RevertOfCulprit struct{}

// GetName returns the name of the rule
func (rule RevertOfCulprit) GetName() string {
	return "RevertOfCulprit"
}

// Run executes the rule.
func (rule RevertOfCulprit) Run(ctx context.Context, ap *AuditParams, rc *RelevantCommit, cs *Clients) (*RuleResult, error) {
	result := &RuleResult{}
	result.RuleResultStatus = RuleFailed

	revert, culprit, err := getRevertAndCulpritChanges(ctx, ap, rc, cs)
	if err != nil {
		return nil, err
	}
	if culprit == nil {
		result.Message = fmt.Sprintf("Commit %q does not appear to be a revert, according to gerrit",
			rc.CommitHash)
		return result, nil
	}

	pr, err := cs.gerrit.IsChangePureRevert(ctx, revert.ChangeID)
	if err != nil {
		return nil, err
	}
	if !pr {
		result.Message = fmt.Sprintf("Commit %q is a revert but not a *pure* revert, according to gerrit",
			rc.CommitHash)
		return result, nil
	}

	// The CommitMessage of the revert must contain the culprit' hash.
	if !strings.Contains(rc.CommitMessage, culprit.CurrentRevision) {
		result.Message = fmt.Sprintf("Commit %q does not include the revision it reverts in its commit message",
			rc.CommitHash)
		return result, nil
	}
	result.RuleResultStatus = RulePassed
	return result, nil
}

// OnlyCommitsOwnChange is a Rule that verifies that commits landed by the
// service account were also authored by that service account.
type OnlyCommitsOwnChange struct{}

// GetName returns the name of the rule
func (rule OnlyCommitsOwnChange) GetName() string {
	return "OnlyCommitsOwnChange"
}

// Run executes the rule.
func (rule OnlyCommitsOwnChange) Run(ctx context.Context, ap *AuditParams, rc *RelevantCommit, cs *Clients) (*RuleResult, error) {
	result := &RuleResult{}
	result.RuleResultStatus = RuleFailed
	if rc.CommitterAccount == ap.TriggeringAccount {
		if rc.CommitterAccount != rc.AuthorAccount {
			result.RuleResultStatus = RuleFailed
			result.Message = fmt.Sprintf("Service account %s committed a commit by someone else: %s",
				rc.CommitterAccount, rc.AuthorAccount)
			return result, nil
		}
	}
	result.RuleResultStatus = RulePassed
	return result, nil
}

func getFailedSteps(commitMessage string) string {
	stepName, err := failedStepFromCommitMessage(commitMessage)
	if err != nil {
		return "compile"
	}
	return stepName
}
