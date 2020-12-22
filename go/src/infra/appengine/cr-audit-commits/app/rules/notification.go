// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cpb "infra/appengine/cr-audit-commits/app/proto"
	"infra/monorail"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/auth"
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
	*cpb.CommentOrFileMonorailIssue
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
		sa := ""
		if signer := auth.GetSigner(ctx); signer != nil {
			info, err := signer.ServiceInfo(ctx)
			if err != nil {
				return "", err
			}
			sa = info.ServiceAccountName
		}

		existingIssue, err := getIssueBySummaryAndAccount(ctx, cfg, summary, sa, cs)
		if err != nil {
			return "", err
		}

		if existingIssue == nil || !isValidIssue(existingIssue, sa, cfg) {
			if issueID, err = PostIssue(ctx, cfg, summary, rc.AuthorAccount, resultText(cfg, rc, false), cs, c.Components, labels); err != nil {
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
	*cpb.FileBugForMergeApprovalViolation
}

// Notify implements Notification.
func (f FileBugForMergeApprovalViolation) Notify(ctx context.Context, cfg *RefConfig, rc *RelevantCommit, cs *Clients, state string) (string, error) {
	return "Purged removed FileBugForMergeApprovalViolation notifications", nil
}

// CommentOnBugToAcknowledgeMerge is used as the notification function of
// merge-ack-rule.
type CommentOnBugToAcknowledgeMerge struct {
	*cpb.CommentOnBugToAcknowledgeMerge
}

// Notify implements Notification.
func (c CommentOnBugToAcknowledgeMerge) Notify(ctx context.Context, cfg *RefConfig, rc *RelevantCommit, cs *Clients, state string) (string, error) {
	return "Purged removed CommentOnBugToAcknowledgeMerge notifications", nil
}

// PostIssue will create an issue based on the given parameters.
func PostIssue(ctx context.Context, cfg *RefConfig, s, o, d string, cs *Clients, components, labels []string) (int32, error) {
	// TODO: Replace monorail v1 api with v3.
	logging.Debugf(ctx, "Attempting to post issue to Monorail for \"%s\"", s)
	labels = append(labels, "Pri-1", "Type-Task")

	// The components for the issue will be the additional components
	// depending on which rules were violated, and the component defined
	// for the repo(if any).
	iss := &monorail.Issue{
		Description: d,
		Components:  components,
		Labels:      labels,
		Status:      monorail.StatusAssigned,
		Summary:     s,
		ProjectId:   cfg.MonorailProject,
	}

	req := &monorail.InsertIssueRequest{
		Issue:     iss,
		SendEmail: true,
	}

	if o != "" {
		logging.Debugf(ctx, "Owner \"%s\" was passed-in; about to insert", o)
		ownAtom := &monorail.AtomPerson{
			Name: o,
		}
		iss.Owner = ownAtom

		resp, err := cs.Monorail.InsertIssue(ctx, req)
		switch status.Code(err) {
		case codes.OK:
			logging.Debugf(ctx, "Successfully filed issue ID %d", resp.Issue.Id)
			return resp.Issue.Id, nil
		case codes.InvalidArgument, codes.Unknown:
			// The Gerrit user doesn't have a corresponding Monorail
			// account so we'll CC them instead. We think that the
			// Monorail V1 API returns HTTP 400 in this case which
			// is mapped to gRPC Invalid Argument or Unknown:
			// https://osscs.corp.google.com/chromium/infra/infra/+/master:go/src/go.chromium.org/luci/grpc/proto/google/rpc/code.proto;l=59;drc=eca556dd94c2c2a42dad90d3f7ee0061885c8242
			// This conflicts with the upstream mapping Internal:
			// https://github.com/grpc/grpc/blob/master/doc/http-grpc-status-mapping.md
			logging.Debugf(ctx, "Failed with error code InvalidArgument or Unknown; attempting to file again with owner set to CC")
			iss.Status = monorail.StatusAvailable
			iss.Owner = nil
			iss.Cc = []*monorail.AtomPerson{ownAtom}
			break // Try to insert again below
		default:
			logging.Debugf(ctx, "Failed with unhandled error \"%v\" with status.Code representation \"%v\"; aborting", err, status.Code(err))
			return 0, err
		}
	}

	logging.Debugf(ctx, "About to insert without owner")
	resp, err := cs.Monorail.InsertIssue(ctx, req)
	if err != nil {
		logging.Debugf(ctx, "Failed with unhandled error \"%v\"; aborting", err)
		return 0, err
	}
	logging.Debugf(ctx, "Successfully filed issue ID %d", resp.Issue.Id)
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
