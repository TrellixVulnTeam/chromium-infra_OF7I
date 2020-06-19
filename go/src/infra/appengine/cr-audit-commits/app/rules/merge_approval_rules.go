// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.chromium.org/luci/common/logging"
)

const (
	mergeApprovedLabel = "Merge-Approved-%s"
)

// OnlyMergeApprovedChange is a Rule that verifies that only approved changes
// are merged into a release branch.
type OnlyMergeApprovedChange struct {
	// AllowedUsers is the list of users who are allowed to author and commit
	// merges.
	AllowedUsers []string
	// AllowedRobots is the list of robot accounts who are allowed to author
	// merges.
	AllowedRobots []string
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
	bugList := strings.Split(bugID, ",")
	milestone := ""
	success := false
	for _, bug := range bugList {
		bugNumber, err := strconv.Atoi(bug)
		if err != nil {
			logging.WithError(err).Errorf(ctx, "Found an invalid bug %s on relevant commit %s", bug, rc.CommitHash)
			continue
		}
		milestone, success = GetToken(ctx, "MilestoneNumber", ap.RepoCfg.Metadata)
		if !success {
			return nil, fmt.Errorf("MilestoneNumber not specified in repository configuration")
		}
		mergeLabel := fmt.Sprintf(mergeApprovedLabel, milestone)
		vIssue, err := issueFromID(ctx, ap.RepoCfg, int32(bugNumber), cs)
		if err != nil {
			logging.WithError(err).Errorf(ctx, "Found an invalid Monorail bug %d on relevant commit %s", bugNumber, rc.CommitHash)
			result.Message = fmt.Sprintf("Revision %s was merged to %s branch and there was an error "+
				"accessing the associated bug (%d):\n\n%s", rc.CommitHash, ap.RepoCfg.BranchName, bugNumber, err.Error())
			return result, nil
		}
		result.MetaData, _ = SetToken(ctx, "BugNumber", strconv.Itoa(int(vIssue.Id)), result.MetaData)
		// Check if the issue has a merge approval label in the comment history
		comments, _ := listCommentsFromIssueID(ctx, ap.RepoCfg, vIssue.Id, cs)
		for _, comment := range comments {
			labels := comment.Updates.Labels
			// Check if the issue has a merge approval label
			for _, label := range labels {
				if label == mergeLabel {
					author := comment.Author.Name
					for _, a := range r.AllowedUsers {
						if author == a {
							result.RuleResultStatus = RulePassed
							return result, nil
						}
					}
					logging.WithError(err).Errorf(ctx, "Found merge approval label %s from a non TPM %s", label, author)
					break
				}
			}
		}
		logging.Errorf(ctx, "Bug %s does not have label %s", bugNumber, mergeLabel)
	}
	result.Message = fmt.Sprintf("Revision %s was merged to %s branch with no merge approval from "+
		"a TPM! \nPlease explain why this change was merged to the branch!", rc.CommitHash, ap.RepoCfg.BranchName)
	return result, nil
}
