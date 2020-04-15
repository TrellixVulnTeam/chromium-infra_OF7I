// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"strconv"
	"strings"

	"context"
	"go.chromium.org/luci/common/logging"
)

const (
	chromeReleaseBotAcc        = "chrome-release-bot@chromium.org"
	chromeReleaseAutoRollerAcc = "chromium-release-autoroll@skia-public.iam.gserviceaccount.com"
	mergeApprovedLabel         = "Merge-Approved-%s"
)

var chromeTPMs = []string{"govind@chromium.org", "govind@google.com",
	"srinivassista@chromium.org", "srinivassista@google.com",
	"bhthompson@chromium.org", "bhthompson@google.com",
	"josafat@chromium.org", "josafat@chromium.org",
	"gkihumba@chromium.org", "gkihumba@google.com",
	"kbleicher@chromium.org", "kbleicher@google.com",
	"mmoss@chromium.org", "mmoss@google.com",
	"benmason@chromium.org", "benmason@google.com",
	"sheriffbot@chromium.org",
	"ketakid@chromium.org", "ketakid@google.com",
	"cindyb@chromium.org", "cindyb@google.com",
	"geohsu@chromium.org", "geohsu@google.com",
	"shawnku@chromium.org", "shawnku@google.com",
	"kariahda@chromium.org", "kariahda@google.com",
	"djmm@chromium.org", "djmm@google.com",
	"pbommana@google.com", "pbommana@chromium.org",
	"bindusuvarna@google.com", "bindusuvarna@chromium.org",
	"dgagnon@chromium.org", "dgagnon@google.com",
	"adetaylor@google.com", "adetaylor@chromium.org"}

// OnlyMergeApprovedChange is a Rule that verifies that only approved changes are merged into a release branch.
type OnlyMergeApprovedChange struct{}

// GetName returns the name of the rule.
func (rule OnlyMergeApprovedChange) GetName() string {
	return "OnlyMergeApprovedChange"
}

// Run executes the rule.
func (rule OnlyMergeApprovedChange) Run(ctx context.Context, ap *AuditParams, rc *RelevantCommit, cs *Clients) (*RuleResult, error) {
	result := &RuleResult{}
	result.RuleResultStatus = ruleFailed

	// Exclude Chrome release bot changes
	if rc.AuthorAccount == chromeReleaseBotAcc || rc.AuthorAccount == chromeReleaseAutoRollerAcc {
		result.RuleResultStatus = rulePassed
		return result, nil
	}

	// Exclude Chrome TPM changes
	for _, tpm := range chromeTPMs {
		if (rc.AuthorAccount == tpm) || (rc.CommitterAccount == tpm) {
			result.RuleResultStatus = rulePassed
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
			logging.WithError(err).Errorf(ctx, "Found an invalid Monorail bug %s on relevant commit %s", bugNumber, rc.CommitHash)
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
					// Check if the author of the merge approval is a TPM
					for _, tpm := range chromeTPMs {
						if author == tpm {
							result.RuleResultStatus = rulePassed
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
