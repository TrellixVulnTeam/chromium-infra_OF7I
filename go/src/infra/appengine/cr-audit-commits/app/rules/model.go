// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"fmt"
	"strings"
	"time"

	ds "go.chromium.org/gae/service/datastore"
)

// AuditStatus is the enum for RelevantCommit.Status.
type AuditStatus int

const (
	// AuditScheduled is the status when an audit is scheduled.
	AuditScheduled AuditStatus = iota
	// AuditCompleted is the status when an audit has been completed.
	AuditCompleted
	// AuditCompletedWithActionRequired is the status when an audit has
	// completed but requires some actions. e.g. notifications.
	AuditCompletedWithActionRequired
	// AuditFailed is the status when an audit has failed.
	AuditFailed
	// AuditPending is the status when some rules may not be decidable yet.
	AuditPending
)

// ToString returns a human-readable version of this status.
func (as AuditStatus) ToString() string {
	switch as {
	case AuditScheduled:
		return "Audit Scheduled"
	case AuditCompleted:
		return "Audited OK"
	case AuditCompletedWithActionRequired:
		return "Action Required"
	case AuditFailed:
		return "Audit Failed"
	default:
		return fmt.Sprintf("Unknown status: %d", int(as))
	}
}

// ColorCode returns a string used to color code the string, as a css class for
// example.
func (as AuditStatus) ColorCode() string {
	switch as {
	case AuditCompleted:
		return "green-status"
	case AuditCompletedWithActionRequired:
		return "red-status"
	case AuditFailed:
		return "red-status"
	default:
		return "normal-status"
	}
}

// ToShortString returns a short string version of this status meant to be used
// as datapoint labels for metrics.
func (as AuditStatus) ToShortString() string {
	switch as {
	case AuditCompleted:
		return "passed"
	case AuditCompletedWithActionRequired:
		return "action required"
	case AuditFailed:
		return "failed"
	case AuditScheduled:
		return "scheduled"
	case AuditPending:
		return "pending recheck"
	default:
		return fmt.Sprintf("unknown:%d", int(as))
	}
}

// RuleStatus is the enum for RuleResult.RuleResultStatus.
type RuleStatus int

const (
	// RuleFailed is the status when a rule has failed.
	RuleFailed RuleStatus = iota
	// RulePassed is the status when a rule has passed.
	RulePassed
	// RuleSkipped is the status when a rule has been skipped.
	RuleSkipped
	// NotificationRequired is the status when a rule requires notifications.
	NotificationRequired
	// RulePending is the status when a rule's result cannot be decided.
	RulePending
	// RuleInvalid is an invalid value, for testing.
	RuleInvalid
)

// ToString returns a human-readable version of this status.
func (rs RuleStatus) ToString() string {
	switch rs {
	case RuleFailed:
		return "Rule Failed"
	case RulePassed:
		return "Rule Passed"
	case RuleSkipped:
		return "Rule Skipped"
	case NotificationRequired:
		return "Notification Required"
	case RulePending:
		return "Rule Pending"
	default:
		return fmt.Sprintf("Unknown status: %d", int(rs))
	}
}

// ColorCode returns a stirng used to color code the string, as a css class for
// example.
func (rs RuleStatus) ColorCode() string {
	switch rs {
	case RuleFailed:
		return "red-status"
	case RulePassed:
		return "green-status"
	default:
		return "normal-status"
	}
}

// RepoState contains the state for each ref we audit. Including
// parameters applicable to dynamically configured refs.
type RepoState struct {
	// RepoURL is expected to point to a branch e.g.
	// https://chromium.googlesource.com/chromium/src.git/+/master
	RepoURL string `gae:"$id"`

	LastKnownCommit        string
	LastKnownCommitTime    time.Time
	LastRelevantCommit     string
	LastRelevantCommitTime time.Time
	// This is the key of the configuration in RuleMap that applies to
	// this git ref. Note that each ref can only be matched to one such
	// configuration.
	ConfigName string

	// These are overridden in the repo config differently for each ref.
	Metadata   string
	BranchName string
}

// RelevantCommit points to a node in a linked list of commits that have
// been considered relevant by CommitScanner.
type RelevantCommit struct {
	RepoStateKey *ds.Key `gae:"$parent"`
	CommitHash   string  `gae:"$id"`

	PreviousRelevantCommit string
	Status                 AuditStatus
	Result                 []RuleResult
	CommitTime             time.Time
	CommitterAccount       string
	AuthorAccount          string
	CommitMessage          string `gae:",noindex"`
	Retries                int32

	// This will catch deprecated fields such as IssueID
	LegacyFields ds.PropertyMap `gae:",extra"`

	// NotifiedAll will be true if all applicable notifications have been
	// processed.
	NotifiedAll bool

	// NotificationStates will have strings of the form `key:value` where
	// the key identifies a specific ruleset that might apply to this
	// commit and value is a freeform string that makes sense to the
	// notification function, used to keep track of the state between
	// retries. e.g. To avoid sending duplicate emails if the notification
	// sends multiple emails and only partially succeeds on the first
	// attempt.
	NotificationStates []string

	// LastExternalPoll records when the commit was last attempted
	// to be audited. This is useful for audit rules that can be left
	// undecided for a period of time, such as TBR auditing in order to
	// limit the frequency of polling external systems.
	LastExternalPoll time.Time
}

// SetResult appends the given RuleResult to the array of results for this commit,
// or update it if one with the same RuleName is already present.
// Returns a boolean indicating whether a change was performed.
func (rc *RelevantCommit) SetResult(rr RuleResult) bool {
	for i, curr := range rc.Result {
		if curr.RuleName == rr.RuleName {
			if curr == rr {
				return false
			}
			rc.Result[i] = rr
			return true
		}

	}
	rc.Result = append(rc.Result, rr)
	return true
}

// RuleResult represents the result of applying a single rule to a commit.
type RuleResult struct {
	RuleName         string
	RuleResultStatus RuleStatus
	Message          string
	// Freeform string that can be used by rules to pass data to notifiers.
	// Notably used by the .GetToken and .SetToken methods.
	MetaData string `gae:",noindex"`
}

// GetViolations returns the subset of RuleResults that are violations.
func (rc *RelevantCommit) GetViolations() []RuleResult {
	violations := []RuleResult{}
	for _, rr := range rc.Result {
		if rr.RuleResultStatus == RuleFailed {
			violations = append(violations, rr)
		}
	}
	return violations
}

// SetNotificationState stores the state for a given rule set.
func (rc *RelevantCommit) SetNotificationState(ruleSetName string, state string) {
	prefix := fmt.Sprintf("%s:", ruleSetName)
	fullTag := fmt.Sprintf("%s:%s", ruleSetName, state)
	for i, v := range rc.NotificationStates {
		if strings.HasPrefix(v, prefix) {
			rc.NotificationStates[i] = fullTag
			return
		}
	}
	rc.NotificationStates = append(rc.NotificationStates, fullTag)
}

// GetNotificationState retrieves the state for a rule set from the
// NotificationStates field.
func (rc *RelevantCommit) GetNotificationState(ruleSetName string) string {
	prefix := fmt.Sprintf("%s:", ruleSetName)
	for _, v := range rc.NotificationStates {
		if strings.HasPrefix(v, prefix) {
			return strings.TrimPrefix(v, prefix)
		}
	}
	return ""
}
