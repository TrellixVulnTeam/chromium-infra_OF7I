// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"net/mail"
	"regexp"

	"go.chromium.org/luci/config/validation"
	cpb "infra/appengine/cr-audit-commits/app/proto"
)

var (
	commitRegex = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)
)

func validateConfig(c *validation.Context, cfg *cpb.Config) {
	for key, refConfig := range cfg.RefConfigs {
		c.Enter("ref_config %s", key)
		validateRefConfig(c, refConfig)
		c.Exit()
	}
}

func validateRefConfig(c *validation.Context, refConfig *cpb.RefConfig) {
	if refConfig.GerritHost == "" {
		c.Errorf("missing gerrit_host")
	}

	if refConfig.GerritRepo == "" {
		c.Errorf("missing gerrit_repo")
	}

	if refConfig.Ref == "" && !refConfig.UseDynamicRefFunc {
		c.Errorf("missing ref")
	}

	if refConfig.StartingCommit == "" {
		if !refConfig.UseDynamicRefFunc {
			c.Errorf("missing starting_commit")
		}
	} else if !commitRegex.MatchString(refConfig.StartingCommit) {
		c.Errorf("invalid starting_commit: %s", refConfig.StartingCommit)
	}

	if refConfig.MonorailProject == "" {
		c.Errorf("missing monorail_project")
	}

	if refConfig.OverwriteLastKnownCommit != "" && !commitRegex.MatchString(refConfig.OverwriteLastKnownCommit) {
		c.Errorf("invalid overwrite_last_known_commit: %s", refConfig.OverwriteLastKnownCommit)
	}

	for key, accountRules := range refConfig.Rules {
		c.Enter("account_rules %s", key)
		validateAccountRules(c, accountRules)
		c.Exit()
	}
}

func validateAccountRules(c *validation.Context, accountRules *cpb.AccountRules) {
	if accountRules.Account == "" {
		c.Errorf("missing account")
	} else if accountRules.Account != "*" {
		if _, err := mail.ParseAddress(accountRules.Account); err != nil {
			c.Errorf("invalid account: %s", err)
		}
	}

	for i, rule := range accountRules.Rules {
		c.Enter("rule #%d", i)
		validateRule(c, rule)
		c.Exit()
	}

	for i, notification := range accountRules.Notifications {
		c.Enter("notification #%d", i)
		validateNotification(c, notification)
		c.Exit()
	}
}

func validateRule(c *validation.Context, rule *cpb.Rule) {
	switch rule.Rule.(type) {
	case *cpb.Rule_AcknowledgeMerge:
	case *cpb.Rule_AutoCommitsPerDay:
	case *cpb.Rule_AutoRevertsPerDay:
	case *cpb.Rule_ChangeReviewed:
		validateChangeReviewed(c, rule.GetChangeReviewed())
	case *cpb.Rule_CulpritAge:
	case *cpb.Rule_CulpritInBuild:
	case *cpb.Rule_FailedBuildIsAppropriateFailure:
	case *cpb.Rule_OnlyCommitsOwnChange:
	case *cpb.Rule_OnlyMergeApprovedChange:
		validateOnlyMergeApprovedChange(c, rule.GetOnlyMergeApprovedChange())
	case *cpb.Rule_OnlyModifiesFilesAndDirsRule:
		validateOnlyModifiesFilesAndDirsRule(c, rule.GetOnlyModifiesFilesAndDirsRule())
	case *cpb.Rule_RevertOfCulprit:
	default:
		c.Errorf("unknown rule")
	}
}

func validateChangeReviewed(c *validation.Context, changeReviewed *cpb.ChangeReviewed) {
	for _, robot := range changeReviewed.Robots {
		if _, err := mail.ParseAddress(robot); err != nil {
			c.Errorf("invalid robot: %s", err)
		}
	}
}

func validateOnlyMergeApprovedChange(c *validation.Context, onlyMergeApprovedChange *cpb.OnlyMergeApprovedChange) {
	for _, user := range onlyMergeApprovedChange.AllowedUsers {
		if _, err := mail.ParseAddress(user); err != nil {
			c.Errorf("invalid allowed_user: %s", err)
		}
	}
	for _, robot := range onlyMergeApprovedChange.AllowedRobots {
		if _, err := mail.ParseAddress(robot); err != nil {
			c.Errorf("invalid allowed_robot: %s", err)
		}
	}
}

func validateOnlyModifiesFilesAndDirsRule(c *validation.Context, onlyModifiesFilesAndDirsRule *cpb.OnlyModifiesFilesAndDirsRule) {
	if onlyModifiesFilesAndDirsRule.Name == "" {
		c.Errorf("missing name")
	}
}

func validateNotification(c *validation.Context, notification *cpb.Notification) {
	switch notification.Notification.(type) {
	case *cpb.Notification_CommentOnBugToAcknowledgeMerge:
	case *cpb.Notification_CommentOrFileMonorailIssue:
		validateCommentOrFileMonorailIssue(c, notification.GetCommentOrFileMonorailIssue())
	case *cpb.Notification_FileBugForMergeApprovalViolation:
		validateFileBugForMergeApprovalViolation(c, notification.GetFileBugForMergeApprovalViolation())
	}
}

func validateCommentOrFileMonorailIssue(c *validation.Context, commentOfFileMonorailIssue *cpb.CommentOrFileMonorailIssue) {
	for _, component := range commentOfFileMonorailIssue.Components {
		if component == "" {
			c.Errorf("empty component")
		}
	}
	for _, label := range commentOfFileMonorailIssue.Labels {
		if label == "" {
			c.Errorf("empty label")
		}
	}
}

func validateFileBugForMergeApprovalViolation(c *validation.Context, fileBugForMergeApprovalViolation *cpb.FileBugForMergeApprovalViolation) {
	for _, component := range fileBugForMergeApprovalViolation.Components {
		if component == "" {
			c.Errorf("empty component")
		}
	}
	for _, label := range fileBugForMergeApprovalViolation.Labels {
		if label == "" {
			c.Errorf("empty label")
		}
	}
}
