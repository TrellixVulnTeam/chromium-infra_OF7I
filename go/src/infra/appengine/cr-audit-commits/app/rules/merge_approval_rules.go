// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	cpb "infra/appengine/cr-audit-commits/app/proto"

	"go.chromium.org/luci/common/logging"
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
	result := &RuleResult{RuleResultStatus: RuleFailed}
	for _, a := range r.AllowedUsers {
		if rc.AuthorAccount == a || rc.CommitterAccount == a {
			result.RuleResultStatus = RulePassed
			return result, nil
		}
	}
	for _, a := range r.AllowedRobots {
		if rc.AuthorAccount == a {
			result.RuleResultStatus = RulePassed
			return result, nil
		}
	}
	bugID, err := bugIDFromCommitMessage(rc.CommitMessage)
	if err != nil {
		result.Message = fmt.Sprintf("Revision %s was merged to %s release branch with no bug attached!"+
			"\nPlease explain why this change was merged to the branch!", rc.CommitHash, ap.RepoCfg.BranchName)
		return result, nil
	}
	// TODO(xinyuoffline): Deduplicate with CommentOnBugToAcknowledgeMerge.Notify().
	bugList := strings.Split(bugID, ",")
	// TODO(jclinton): figure out if we still need this
	// milestone := ""
	// success := false
	for _, bug := range bugList {
		bugNumber, err := strconv.Atoi(bug)
		if err != nil {
			// TODO(xinyuoffline): Is this an error?
			logging.WithError(err).Warningf(ctx, "OnlyMergeApprovedChange: Found an invalid bug %s on relevant commit %s", bug, rc.CommitHash)
			continue
		}
		// TODO(xinyuoffline): Calculate this up front outside of the loop.
		// TODO(jclinton): figure out if we still need this
		// milestone, success = GetToken(ctx, "MilestoneNumber", ap.RepoCfg.Metadata)
		// if !success {
		// 	return nil, errors.New("MilestoneNumber not specified in repository configuration")
		// }
		// mergeLabel := fmt.Sprintf(mergeApprovedLabel, milestone)
		vIssue, err := issueFromID(ctx, ap.RepoCfg, int32(bugNumber), cs)
		if err != nil {
			result.Message = fmt.Sprintf(
				"Revision %s was merged to %s branch and there was an error "+
					"accessing the associated bug https://bugs.chromium.org/p/%s/issues/detail?id=%d:\n%s",
				rc.CommitHash, ap.RepoCfg.BranchName, ap.RepoCfg.MonorailProject, bugNumber, err)
			return result, nil
		}
		result.MetaData, _ = SetToken(ctx, "BugNumber", strconv.Itoa(int(vIssue.Id)), result.MetaData)

		// TODO(jclinton) decide if we should retain this check
		// For now, mark as passed regardles of comments & labels

		// Check if the issue has a merge approval label in the comment history
		// comments, _ := listCommentsFromIssueID(ctx, ap.RepoCfg, vIssue.Id, cs)
		// for _, comment := range comments {
		// 	labels := comment.Updates.Labels
		// 	// Check if the issue has a merge approval label
		// 	for _, label := range labels {
		// 		if label == mergeLabel {
		// 			author := comment.Author.Name
		// 			for _, a := range r.AllowedUsers {
		// 				if author == a {
		// 					result.RuleResultStatus = RulePassed
		// 					return result, nil
		// 				}
		// 			}
		// 			// TODO(xinyuoffline): Is this an error?
		// 			logging.WithError(err).Warningf(ctx, "OnlyMergeApprovedChange: Found merge approval label %s from a non TPM %s", label, author)
		// 			break
		// 		}
		// 	}
		// }
		// TODO(xinyuoffline): Is this an error?
		// logging.Warningf(ctx, "OnlyMergeApprovedChange: https://bugs.chromium.org/p/%s/issues/detail?id=%d does not have label %s", ap.RepoCfg.MonorailProject, bugNumber, mergeLabel)

		result.RuleResultStatus = RulePassed
		return result, nil
	}
	result.Message = fmt.Sprintf("Revision %s was merged to %s branch with no merge approval from "+
		"a TPM! \nPlease explain why this change was merged to the branch!", rc.CommitHash, ap.RepoCfg.BranchName)
	return result, nil
}
