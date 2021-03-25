// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"strings"
	"time"

	cpb "infra/appengine/cr-audit-commits/app/proto"
	"infra/appengine/cr-audit-commits/app/rules"
)

const (
	// StuckScannerDuration refers how many hours after a ref stops auditing,
	// a bug will be filed.
	StuckScannerDuration = time.Duration(24) * time.Hour

	// MaxCommitsPerRefUpdate is the maximum commits that the Gitiles git.Log
	// API should return every time it is called.
	MaxCommitsPerRefUpdate = 6000

	monorailAPIURL = "https://monorail-prod.appspot.com/_ah/api/monorail/v1"
)

// getGerritURL converts proto's GerritHost to a gerrit url.
func getGerritURL(gerritHost string) string {
	gerritHostSlices := strings.Split(gerritHost, ".")
	gerritHostSlices[0] = "https://" + gerritHostSlices[0] + "-review"
	return strings.Join(gerritHostSlices, ".")
}

// getAccountRules converts proto's AccountRules to rules' AccountRules.
func getAccountRules(protoAccountRules map[string]*cpb.AccountRules) map[string]rules.AccountRules {
	accountRules := make(map[string]rules.AccountRules, len(protoAccountRules))
	for k, v := range protoAccountRules {
		// TODO: Currently the rules.AccountRules only contains 1 notification
		// function, so here I only take the first notification function in the
		// config. Will alter the logic when using multiple notification
		// functions.
		var notification rules.Notification
		if len(v.Notifications) > 0 {
			switch v.Notifications[0].Notification.(type) {
			case *cpb.Notification_CommentOnBugToAcknowledgeMerge:
				notification = rules.CommentOnBugToAcknowledgeMerge{}
			case *cpb.Notification_CommentOrFileMonorailIssue:
				notification = rules.CommentOrFileMonorailIssue{
					CommentOrFileMonorailIssue: v.Notifications[0].GetCommentOrFileMonorailIssue(),
				}
			case *cpb.Notification_FileBugForMergeApprovalViolation:
				notification = rules.FileBugForMergeApprovalViolation{
					FileBugForMergeApprovalViolation: v.Notifications[0].GetFileBugForMergeApprovalViolation(),
				}
			}
		}

		var rs []rules.Rule
		for _, r := range v.Rules {
			switch r.Rule.(type) {
			case *cpb.Rule_AcknowledgeMerge:
				rs = append(rs, rules.AcknowledgeMerge{})
			case *cpb.Rule_AutoCommitsPerDay:
				rs = append(rs, rules.AutoCommitsPerDay{})
			case *cpb.Rule_AutoRevertsPerDay:
				rs = append(rs, rules.AutoRevertsPerDay{})
			case *cpb.Rule_ChangeReviewed:
				rs = append(rs, rules.ChangeReviewed{
					ChangeReviewed: r.GetChangeReviewed(),
				})
			case *cpb.Rule_CulpritAge:
				rs = append(rs, rules.CulpritAge{})
			case *cpb.Rule_CulpritInBuild:
				rs = append(rs, rules.CulpritInBuild{})
			case *cpb.Rule_FailedBuildIsAppropriateFailure:
				rs = append(rs, rules.FailedBuildIsAppropriateFailure{})
			case *cpb.Rule_OnlyCommitsOwnChange:
				rs = append(rs, rules.OnlyCommitsOwnChange{})
			case *cpb.Rule_OnlyMergeApprovedChange:
				rs = append(rs, rules.OnlyMergeApprovedChange{
					OnlyMergeApprovedChange: r.GetOnlyMergeApprovedChange(),
				})
			case *cpb.Rule_OnlyModifiesFilesAndDirsRule:
				rs = append(rs, rules.OnlyModifiesFilesAndDirsRule{
					OnlyModifiesFilesAndDirsRule: r.GetOnlyModifiesFilesAndDirsRule(),
				})
			case *cpb.Rule_RevertOfCulprit:
				rs = append(rs, rules.RevertOfCulprit{})
			}
		}

		accountRules[k] = rules.AccountRules{
			Account:      v.Account,
			Notification: notification,
			Rules:        rs,
		}
	}
	return accountRules
}

// GetUpdatedRuleMap returns a map of each monitored repository to a list of
// account/rules structs.
func GetUpdatedRuleMap(c context.Context) map[string]*rules.RefConfig {
	configs := Get(c).RefConfigs
	updatedRuleMap := make(map[string]*rules.RefConfig, len(configs))

	// Use configs from LUCI-config service to get a ruleMap.
	for k, refConfig := range configs {
		var dynamicRefFunc rules.DynamicRefFunc
		if refConfig.UseDynamicRefFunc {
			dynamicRefFunc = rules.ReleaseConfig
		}

		updatedRuleMap[k] = &rules.RefConfig{
			BaseRepoURL:    "https://" + refConfig.GerritHost + "/" + refConfig.GerritRepo,
			GerritURL:      getGerritURL(refConfig.GerritHost),
			BranchName:     refConfig.Ref,
			StartingCommit: refConfig.StartingCommit,
			// TODO: For test environment, the MonorailAPIURL should be different.
			MonorailAPIURL:           monorailAPIURL,
			MonorailProject:          refConfig.MonorailProject,
			Rules:                    getAccountRules(refConfig.Rules),
			OverwriteLastKnownCommit: refConfig.OverwriteLastKnownCommit,
			DynamicRefFunction:       dynamicRefFunc,
		}
	}

	return updatedRuleMap
}
