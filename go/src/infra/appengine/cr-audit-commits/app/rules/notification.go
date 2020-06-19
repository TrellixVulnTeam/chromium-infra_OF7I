// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"context"

	"go.chromium.org/gae/service/info"
	"go.chromium.org/luci/common/logging"

	"infra/monorail"
)

// Notification is a type that needs to be implemented by functions
// intended to notify about violations in rules.
//
// The notification function is expected to determine if there is a violation
// by checking the results of calling .GetViolations() on the RelevantCommit
// and not just blindly send a notification.
//
// The state parameter is expected to be used to keep the state between retries
// to avoid duplicating notifications, its value will be either the empty string
// or the first element of the return value of a previous call to this function
// for the same commit.
//
// e.g. Return ('notificationSent', nil) if everything goes well, and if the
// incoming state already equals 'notificationSent', then don't send the
// notification, as that would indicate that a previous call already took care
// of that. The state string is a short freeform string that only needs to be
// understood by the NotificationFunc itself, and should exclude colons (`:`).
type Notification interface {
	Notify(ctx context.Context, cfg *RefConfig, rc *RelevantCommit, cs *Clients, state string) (string, error)
}

// CommentOrFileMonorailIssue files a bug to Monorail.
//
// It checks if the failure has already been reported to Monorail and files a
// new bug if it hasn't. If a bug already exists this function will try to add
// a comment and associate it to the bug.
type CommentOrFileMonorailIssue struct {
	Components []string
	Labels     []string
}

// Notify implements Notification.
func (c CommentOrFileMonorailIssue) Notify(ctx context.Context, cfg *RefConfig, rc *RelevantCommit, cs *Clients, state string) (string, error) {
	summary := fmt.Sprintf("Audit violation detected on %q", rc.CommitHash)
	// Make sure that at least one of the rules that were violated had
	// .FileBug set to true.
	violations := rc.GetViolations()
	fileBug := len(violations) > 0
	labels := append([]string{"Restrict-View-Google"}, c.Labels...)
	if fileBug && state == "" {
		issueID := int32(0)
		sa, err := info.ServiceAccount(ctx)
		if err != nil {
			return "", err
		}

		existingIssue, err := getIssueBySummaryAndAccount(ctx, cfg, summary, sa, cs)
		if err != nil {
			return "", err
		}

		if existingIssue == nil || !isValidIssue(existingIssue, sa, cfg) {
			if issueID, err = PostIssue(ctx, cfg, summary, resultText(cfg, rc, false), cs, c.Components, labels); err != nil {
				return "", err
			}
		} else {
			// The issue exists and is valid, but it's not
			// associated with the datastore entity for this commit.
			issueID = existingIssue.Id
			if err = postComment(ctx, cfg, existingIssue.Id, resultText(cfg, rc, true), cs, labels); err != nil {
				return "", err
			}
		}
		state = fmt.Sprintf("BUG=%d", issueID)
	}
	return state, nil
}

// FileBugForMergeApprovalViolation is the notification function for
// merge-approval-rules.
type FileBugForMergeApprovalViolation struct {
	Components []string
	Labels     []string
}

// Notify implements Notification.
func (f FileBugForMergeApprovalViolation) Notify(ctx context.Context, cfg *RefConfig, rc *RelevantCommit, cs *Clients, state string) (string, error) {
	milestone, ok := GetToken(ctx, "MilestoneNumber", cfg.Metadata)
	if !ok {
		return "", fmt.Errorf("MilestoneNumber not specified in repository configuration")
	}
	labels := append([]string{fmt.Sprintf("M-%s", milestone)}, f.Labels...)
	for _, result := range rc.Result {
		if result.RuleResultStatus != RuleFailed {
			continue
		}
		bug, success := GetToken(ctx, "BugNumber", result.MetaData)
		if state == "" {
			// Comment on the bug if any. If not, file a new bug.
			if success {
				bugID, _ := strconv.Atoi(bug)
				err := postComment(ctx, cfg, int32(bugID), resultText(cfg, rc, true), cs, labels)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("Comment posted on BUG=%d", int32(bugID)), nil
			}
			c := CommentOrFileMonorailIssue{
				Components: f.Components,
				Labels:     labels,
			}
			return c.Notify(ctx, cfg, rc, cs, state)
		}
	}
	return "No violation found", nil
}

// CommentOnBugToAcknowledgeMerge is used as the notification function of
// merge-ack-rule.
type CommentOnBugToAcknowledgeMerge struct{}

// Notify implements Notification.
func (c CommentOnBugToAcknowledgeMerge) Notify(ctx context.Context, cfg *RefConfig, rc *RelevantCommit, cs *Clients, state string) (string, error) {
	milestone, ok := GetToken(ctx, "MilestoneNumber", cfg.Metadata)
	if !ok {
		return "", fmt.Errorf("MilestoneNumber not specified in repository configuration")
	}
	branchName := strings.Replace(cfg.BranchName, "refs/branch-heads/", "", -1)
	mergeAckLabel := fmt.Sprintf("Merge-Merged-%s-%s", milestone, branchName)
	mergeLabel := fmt.Sprintf("-Merge-Approved-%s", milestone)
	labels := []string{mergeLabel, mergeAckLabel}
	for _, result := range rc.Result {
		if result.RuleResultStatus != NotificationRequired {
			continue
		}
		bugID, success := GetToken(ctx, "BugNumbers", result.MetaData)
		if state == "" {
			if success {
				logging.Infof(ctx, "Found bug(s): '%s' on relevant commit %s", bugID, rc.CommitHash)
				bugList := strings.Split(bugID, ",")
				validBugs := ""
				for _, bug := range bugList {
					bugNumber, err := strconv.Atoi(bug)
					if err != nil {
						logging.WithError(err).Errorf(ctx, "Found an invalid bug %s on relevant commit %s", bug, rc.CommitHash)
						continue
					}
					vIssue, err := issueFromID(ctx, cfg, int32(bugNumber), cs)
					if err != nil {
						logging.WithError(err).Errorf(ctx, "Found an invalid Monorail bug %d on relevant commit %s", bugNumber, rc.CommitHash)
						continue
					}
					mergeAckComment := "The following revision refers to this bug: \n%s\n\nCommit: %s\nAuthor: %s\nCommiter: %s\nDate: %s\n\n%s"
					comment := fmt.Sprintf(mergeAckComment, cfg.LinkToCommit(rc.CommitHash), rc.CommitHash, rc.AuthorAccount, rc.CommitterAccount, rc.CommitTime, rc.CommitMessage)
					if err = postComment(ctx, cfg, int32(vIssue.Id), comment, cs, labels); err != nil {
						logging.Errorf(ctx, "Could not comment on bug %s", bug)
						continue
					}
					if validBugs == "" {
						validBugs = bug
					} else {
						validBugs += fmt.Sprintf(",%s", bug)
					}
				}
				if validBugs != "" {
					return fmt.Sprintf("Comment posted on BUG(S)=%s", validBugs), nil
				}
			}
			return "", fmt.Errorf("No bug found or could not comment on bug(s) found on revision %s", rc.CommitHash)
		}
	}
	return "No notification required", nil
}

// PostIssue will create an issue based on the given parameters.
func PostIssue(ctx context.Context, cfg *RefConfig, s, d string, cs *Clients, components, labels []string) (int32, error) {
	// The components for the issue will be the additional components
	// depending on which rules were violated, and the component defined
	// for the repo(if any).
	iss := &monorail.Issue{
		Description: d,
		Components:  components,
		Labels:      labels,
		Status:      monorail.StatusUntriaged,
		Summary:     s,
		ProjectId:   cfg.MonorailProject,
	}

	req := &monorail.InsertIssueRequest{
		Issue:     iss,
		SendEmail: true,
	}

	resp, err := cs.Monorail.InsertIssue(ctx, req)
	if err != nil {
		return 0, err
	}
	return resp.Issue.Id, nil
}

// isValidIssue checks that the monorail issue was created by the app and
// has the correct summary. This is to avoid someone
// suppressing an audit alert by creating a spurious bug.
func isValidIssue(iss *monorail.Issue, sa string, cfg *RefConfig) bool {
	for _, st := range []string{
		monorail.StatusFixed,
		monorail.StatusVerified,
		monorail.StatusDuplicate,
		monorail.StatusWontFix,
		monorail.StatusArchived,
	} {
		if iss.Status == st {
			// Issue closed, file new one.
			return false
		}
	}
	if strings.HasPrefix(iss.Summary, "Audit violation detected on") && iss.Author.Name == sa {
		return true
	}
	return false
}

func getIssueBySummaryAndAccount(ctx context.Context, cfg *RefConfig, s, a string, cs *Clients) (*monorail.Issue, error) {
	q := fmt.Sprintf("summary:\"%s\" reporter:\"%s\"", s, a)
	req := &monorail.IssuesListRequest{
		ProjectId: cfg.MonorailProject,
		Can:       monorail.IssuesListRequest_ALL,
		Q:         q,
	}
	resp, err := cs.Monorail.IssuesList(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, iss := range resp.Items {
		if iss.Summary == s {
			return iss, nil
		}
	}
	return nil, nil
}

func postComment(ctx context.Context, cfg *RefConfig, iID int32, c string, cs *Clients, labels []string) error {
	req := &monorail.InsertCommentRequest{
		Comment: &monorail.InsertCommentRequest_Comment{
			Content: c,
			Updates: &monorail.Update{
				Labels: labels,
			},
		},
		Issue: &monorail.IssueRef{
			IssueId:   iID,
			ProjectId: cfg.MonorailProject,
		},
	}
	_, err := cs.Monorail.InsertComment(ctx, req)
	return err
}

func resultText(cfg *RefConfig, rc *RelevantCommit, issueExists bool) string {
	sort.Slice(rc.Result, func(i, j int) bool {
		if rc.Result[i].RuleResultStatus == rc.Result[j].RuleResultStatus {
			return rc.Result[i].RuleName < rc.Result[j].RuleName
		}
		return rc.Result[i].RuleResultStatus < rc.Result[j].RuleResultStatus
	})
	rows := []string{}
	for _, rr := range rc.Result {
		rows = append(rows, fmt.Sprintf(" - %s: %s -- %s", rr.RuleName, rr.RuleResultStatus.ToString(), rr.Message))
	}

	results := fmt.Sprintf("Here's a summary of the rules that were executed: \n%s",
		strings.Join(rows, "\n"))

	if issueExists {
		return results
	}

	description := "An audit of the git commit at %q found at least one violation. \n" +
		" The commit was created by %s and committed by %s.\n\n%s"
	return fmt.Sprintf(description, cfg.LinkToCommit(rc.CommitHash), rc.AuthorAccount, rc.CommitterAccount, results)
}
