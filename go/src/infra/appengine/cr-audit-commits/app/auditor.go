// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/golang/protobuf/ptypes"
	ds "go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/api/gitiles"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/proto/git"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/server/router"

	"infra/appengine/cr-audit-commits/app/config"
	"infra/appengine/cr-audit-commits/app/rules"
)

// Tests can put mock clients here, prod code will ignore this global.
var auditorTestClients *rules.Clients

// Auditor is the main entry point for scanning the commits on a given ref and
// auditing those that are relevant to the configuration.
//
// It scans the ref in the repo and creates entries for relevant commits,
// executes the audit functions on such commits, and calls notification
// functions when appropriate.
//
// This is expected to run every 5 minutes and for that reason, it has designed
// to stop 4 minutes 30 seconds and save any partial progress.
func Auditor(rc *router.Context) {
	outerCtx, resp := rc.Context, rc.Writer

	// Create a derived context with a 4:30 timeout s.t. we have enough
	// time to save results for at least some of the audited commits,
	// considering that a run of this handler will be scheduled every 5
	// minutes.
	ctx, cancelInnerCtx := context.WithTimeout(outerCtx, time.Second*time.Duration(4*60+30))
	defer cancelInnerCtx()

	cfg, repoState, err := loadConfig(rc.Context, rc.Request.FormValue("refUrl"))
	if err != nil {
		http.Error(resp, err.Error(), 400)
		return
	}

	var cs *rules.Clients
	if auditorTestClients != nil {
		cs = auditorTestClients
	} else {
		httpClient, err := rules.GetAuthenticatedHTTPClient(ctx, rules.GerritScope, rules.EmailScope)
		if err != nil {
			http.Error(resp, err.Error(), 500)
			return
		}

		cs, err = rules.InitializeClients(ctx, cfg, httpClient)
		if err != nil {
			http.Error(resp, err.Error(), 500)
			return
		}
	}

	fl, err := getCommitLog(ctx, cfg, repoState, cs)
	if err != nil {
		http.Error(resp, err.Error(), 502)
		return
	}

	// Iterate over the log, creating relevantCommit entries as appropriate
	// and updating repoState. If the context expires during this process,
	// save the repoState and bail.
	err = scanCommits(ctx, fl, cfg, repoState)
	if err != nil && err != context.DeadlineExceeded {
		logging.WithError(err).Errorf(ctx, "Could not save new relevant commit")
		http.Error(resp, err.Error(), 503)
		return
	}
	// Save progress with an unexpired context.
	if putErr := ds.Put(outerCtx, repoState); putErr != nil {
		logging.WithError(putErr).Errorf(outerCtx, "Could not save last known/interesting commits")
		http.Error(resp, putErr.Error(), 503)
		return
	}
	if err == context.DeadlineExceeded {
		// If the context has expired do not proceed with auditing.
		return
	}

	// TODO(crbug.com/976447): Split the auditing onto its own cron schedule.
	//
	// Send the relevant commits to workers to be audited, note that this
	// doesn't persist the changes, because we want to persist them together
	// in a single transaction for efficiency.
	//
	// If the context expires while performing the audits, save the commits
	// that were audited and bail.
	auditedCommits, err := performScheduledAudits(ctx, cfg, repoState, cs)
	if err != nil && err != context.DeadlineExceeded {
		http.Error(resp, err.Error(), 500)
		return
	}
	if putErr := saveAuditedCommits(outerCtx, auditedCommits, cfg, repoState); putErr != nil {
		http.Error(resp, err.Error(), 503)
		return
	}
	if err == context.DeadlineExceeded {
		// If the context has expired do not proceed with notifications.
		return
	}

	err = notifyAboutViolations(ctx, cfg, repoState, cs)
	if err != nil {
		http.Error(resp, err.Error(), 502)
		return
	}

}

// getCommitLog gets from gitiles a list from the tip to the last known commit
// of the ref in reverse chronological (as per git parentage) order.
func getCommitLog(ctx context.Context, cfg *rules.RefConfig, repoState *rules.RepoState, cs *rules.Clients) ([]*git.Commit, error) {

	host, project, err := gitiles.ParseRepoURL(cfg.BaseRepoURL)
	if err != nil {
		return []*git.Commit{}, err
	}

	gc, err := cs.NewGitilesClient(host)
	if err != nil {
		logging.WithError(err).Errorf(ctx, "Could not create new gitiles client for %s", host)
		return []*git.Commit{}, err
	}

	// Get the tip of the repo
	branchName := cfg.BranchName
	if !strings.HasPrefix(branchName, "refs/heads") {
		branchName = "refs/heads/" + branchName
	}
	logging.Debugf(ctx, "branchName: %s", branchName)
	resp, err := gc.Refs(ctx, &gitilespb.RefsRequest{Project: project, RefsPath: branchName})
	if err != nil {
		logging.WithError(err).Errorf(ctx, "Could not get the tip of the ref %s", project)
		return []*git.Commit{}, err
	}
	newHead, ok := resp.Revisions[fmt.Sprintf("%s/%s", branchName, branchName)]
	if !ok {
		return []*git.Commit{},
			fmt.Errorf("Could not get the branch %s in ref %s", branchName, project)
	}
	oldHead := repoState.LastKnownCommit

	logReq := &gitilespb.LogRequest{
		Project:            project,
		ExcludeAncestorsOf: oldHead,
		Committish:         newHead,
	}

	fl, err := gitiles.PagingLog(ctx, gc, logReq, 6000)
	switch status.Code(err) {
	case codes.OK:
		return fl, nil
	case codes.NotFound:
		// Handled below
		break
	default:
		// Gitiles accidental error
		logging.WithError(err).Errorf(ctx, "Could not get children of revision %s from gitiles",
			oldHead)
		return []*git.Commit{}, err
	}

	// Either:
	//  (1) oldHead is no longer known in gitiles (force push),
	//  (2) newHead is no longer known in gitiles (eventual consistency,
	//     or concurrent force push executed just now, or ACLs change)
	//  (3) gitiles accidental 404, aka fluke.
	// We can assume that case (1) should be an extreme case. When it appears,
	// it is acceptable for Audit App to ignore some commits and start further
	// auditing directly from newHead.
	// In cases (2) and (3), retries should clear the problem, while (1) we
	// should handle now.
	// Reference: https://source.chromium.org/chromium/infra/infra/+/master:go/src/go.chromium.org/luci/scheduler/appengine/task/gitiles/gitiles.go;drc=d4602d5e3619fed71b1a22ae426580d7ffb7fc87;l=425?originalUrl=https:%2F%2Fcs.chromium.org%2F

	// Fetch log of newHead only
	var newErr error
	fl, newErr = gitiles.PagingLog(ctx, gc, &gitilespb.LogRequest{
		Project:    project,
		Committish: newHead,
	}, 1)
	if newErr != nil {
		// case (2) or (3)
		logging.WithError(newErr).Errorf(ctx, "Could not get gitiles log from revision %s", newHead)
		return []*git.Commit{}, newErr
	}

	// Fetch log of oldHead only
	_, oldErr := gitiles.PagingLog(ctx, gc, &gitilespb.LogRequest{
		Project:    project,
		Committish: oldHead,
	}, 1)
	switch status.Code(oldErr) {
	case codes.NotFound:
		// case (1)
		logging.Infof(ctx, "Force push detected; start auditing from new head %s", newHead)
		return fl, nil
	case codes.OK:
		return []*git.Commit{},
			fmt.Errorf("Weirdly, log(%s) and log(%s) work, but not log(%s..%s)",
				oldHead, newHead, oldHead, newHead)
	default:
		// case (3)
		logging.WithError(oldErr).Errorf(ctx, "Could not get gitiles log from revision %s", oldHead)
		return []*git.Commit{}, oldErr
	}
}

// scanCommits iterates over the list of commits in the given log, decides if
// each is relevant to any of the configured rulesets and creates records for
// each that is. Also updates the record for the ref, but does not persist it,
// this is instead done by Auditor after this function is executed. This is left
// to the handler in case the context given to this function expires before
// reaching the end of the log.
func scanCommits(ctx context.Context, fl []*git.Commit, cfg *rules.RefConfig, repoState *rules.RepoState) error {
	// Iterate the commit list in a chronological order.
	// TODO(crbug/1112597): Time out problem. Suppose we have multiple branches
	// and the context reaches the deadline before all the commits are scanned
	// and saved in database, the LastKnownCommit could be a commit on one of
	// the branch. In the next round, it is possible that some commits are
	// scanned again.
	for i := len(fl) - 1; i >= 0; i-- {
		commit := fl[i]
		relevant := false
		for _, ruleSet := range cfg.Rules {
			if ruleSet.MatchesCommit(commit) {
				n, err := saveNewRelevantCommit(ctx, repoState, commit)
				if err != nil {
					return err
				}
				repoState.LastRelevantCommit = n.CommitHash
				repoState.LastRelevantCommitTime = n.CommitTime
				// If the commit matches one ruleSet that's
				// enough. Break to move on to the next commit.
				relevant = true
				break
			}
		}
		ScannedCommits.Add(ctx, 1, relevant, repoState.ConfigName)
		repoState.LastKnownCommit = commit.Id
		// Ignore possible error, this time is used for display purposes only.
		if commit.Committer != nil {
			ct, _ := ptypes.Timestamp(commit.Committer.Time)
			repoState.LastKnownCommitTime = ct
		}

	}
	return nil
}

func saveNewRelevantCommit(ctx context.Context, state *rules.RepoState, commit *git.Commit) (*rules.RelevantCommit, error) {
	rk := ds.KeyForObj(ctx, state)

	commitTime, err := ptypes.Timestamp(commit.GetCommitter().GetTime())
	if err != nil {
		logging.WithError(err).Errorf(ctx, "Invalid commit time: %s", commitTime)
		return nil, err
	}
	rc := &rules.RelevantCommit{
		RepoStateKey:           rk,
		CommitHash:             commit.Id,
		PreviousRelevantCommit: state.LastRelevantCommit,
		Status:                 rules.AuditScheduled,
		CommitTime:             commitTime,
		CommitterAccount:       commit.Committer.Email,
		AuthorAccount:          commit.Author.Email,
		CommitMessage:          commit.Message,
	}

	if err := ds.Put(ctx, rc, state); err != nil {
		logging.WithError(err).Errorf(ctx, "Could not save %s", rc.CommitHash)
		return nil, err
	}
	logging.Infof(ctx, "saved %s", rc)

	return rc, nil
}

// saveAuditedCommits transactionally saves the records for the commits that
// were audited.
func saveAuditedCommits(ctx context.Context, auditedCommits map[string]*rules.RelevantCommit, cfg *rules.RefConfig, repoState *rules.RepoState) error {
	// We will read the relevant commits into this slice before modifying
	// them, to ensure that we don't overwrite changes that may have been
	// saved to the datastore between the time the query in performScheduled
	// audits ran and the beginning of the transaction below; as may have
	// happened if two runs of the Audit handler ran in parallel.
	originalCommits := []*rules.RelevantCommit{}
	for _, auditedCommit := range auditedCommits {
		originalCommits = append(originalCommits, &rules.RelevantCommit{
			CommitHash:   auditedCommit.CommitHash,
			RepoStateKey: auditedCommit.RepoStateKey,
		})
	}

	// We save all the results produced by the workers in a single
	// transaction. We do it this way because there is rate limit of 1 QPS
	// in a single entity group. (All relevant commits for a single repo
	// are contained in a single entity group)
	return ds.RunInTransaction(ctx, func(ctx context.Context) error {
		commitsToPut := make([]*rules.RelevantCommit, 0, len(auditedCommits))
		if err := ds.Get(ctx, originalCommits); err != nil {
			return err
		}
		for _, currentCommit := range originalCommits {
			if auditedCommit, ok := auditedCommits[currentCommit.CommitHash]; ok {
				// Only save those that are still not in a decided
				// state in the datastore to avoid racing a possible
				// parallel run of this handler.
				if currentCommit.Status == rules.AuditScheduled || currentCommit.Status == rules.AuditPending {
					commitsToPut = append(commitsToPut, auditedCommit)
				}
			}
		}
		if err := ds.Put(ctx, commitsToPut); err != nil {
			return err
		}
		for _, c := range commitsToPut {
			if c.Status != rules.AuditScheduled {
				AuditedCommits.Add(ctx, 1, c.Status.ToShortString(), repoState.ConfigName)
			}
		}
		return nil
	}, nil)
}

// notifyAboutViolations is meant to notify about violations to audit
// policies by calling the notification functions registered for each ruleSet
// that matches a commit in the AuditCompletedWithActionRequired status.
func notifyAboutViolations(ctx context.Context, cfg *rules.RefConfig, repoState *rules.RepoState, cs *rules.Clients) error {

	cfgk := ds.KeyForObj(ctx, repoState)

	cq := ds.NewQuery("RelevantCommit").Ancestor(cfgk).Eq("Status", rules.AuditCompletedWithActionRequired).Eq("NotifiedAll", false)
	err := ds.Run(ctx, cq, func(rc *rules.RelevantCommit) error {
		errors := []error{}
		var err error
		for ruleSetName, ruleSet := range cfg.Rules {
			if ruleSet.MatchesRelevantCommit(rc) {
				state := rc.GetNotificationState(ruleSetName)
				state, err = ruleSet.Notification.Notify(ctx, cfg, rc, cs, state)
				if err == context.DeadlineExceeded {
					return err
				} else if err != nil {
					errors = append(errors, err)
				}
				rc.SetNotificationState(ruleSetName, state)
			}
		}
		if len(errors) == 0 {
			rc.NotifiedAll = true
		}
		for _, e := range errors {
			logging.WithError(e).Errorf(ctx, "Failed notification for detected violation on %s.",
				cfg.LinkToCommit(rc.CommitHash))
			NotificationFailures.Add(ctx, 1, "Violation", repoState.ConfigName)
		}
		err = ds.Put(ctx, rc)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	cq = ds.NewQuery("RelevantCommit").Ancestor(cfgk).Eq("Status", rules.AuditFailed).Eq("NotifiedAll", false)
	return ds.Run(ctx, cq, func(rc *rules.RelevantCommit) error {
		err := reportAuditFailure(ctx, cfg, rc, cs)

		if err == nil {
			rc.NotifiedAll = true
			err = ds.Put(ctx, rc)
			if err != nil {
				logging.WithError(err).Errorf(ctx, "Failed to save notification state for failed audit on %s.",
					cfg.LinkToCommit(rc.CommitHash))
				NotificationFailures.Add(ctx, 1, "AuditFailure", repoState.ConfigName)
			}
		} else {
			logging.WithError(err).Errorf(ctx, "Failed to file bug for audit failure on %s.", cfg.LinkToCommit(rc.CommitHash))
			NotificationFailures.Add(ctx, 1, "AuditFailure", repoState.ConfigName)
		}
		return nil
	})
}

// reportAuditFailure is meant to file a bug about a revision that has
// repeatedly failed to be audited. i.e. one or more rules return errors on
// each run.
//
// This does not necessarily mean that a policy has been violated, but only
// that the audit app has not been able to determine whether one exists. One
// such failure could be due to a bug in one of the rules or an error in one of
// the services we depend on (monorail, gitiles, gerrit).
func reportAuditFailure(ctx context.Context, cfg *rules.RefConfig, rc *rules.RelevantCommit, cs *rules.Clients) error {
	summary := fmt.Sprintf("Audit on %q failed over %d times", rc.CommitHash, rc.Retries)
	description := fmt.Sprintf("commit %s has caused the audit process to fail repeatedly, "+
		"please audit by hand and don't close this bug until the root cause of the failure has been "+
		"identified and resolved.", cfg.LinkToCommit(rc.CommitHash))

	var err error
	issueID := int32(0)
	issueID, err = rules.PostIssue(ctx, cfg, summary, description, cs, []string{"Infra>Security>Audit"}, []string{"AuditFailure"})
	if err == nil {
		rc.SetNotificationState("AuditFailure", fmt.Sprintf("BUG=%d", issueID))
		// Do not sent further notifications for this commit. This needs
		// to be audited by hand.
		rc.NotifiedAll = true
	}
	return err
}

// loadConfig returns both repository status and config based on given refURL.
func loadConfig(ctx context.Context, refURL string) (*rules.RefConfig, *rules.RepoState, error) {
	rs := &rules.RepoState{RepoURL: refURL}
	err := ds.Get(ctx, rs)
	if err != nil {
		return nil, nil, err
	}

	cfg, ok := config.GetRuleMap()[rs.ConfigName]
	if !ok {
		return nil, nil, fmt.Errorf("Unknown or missing config %s", rs.ConfigName)
	}
	return cfg.SetConcreteRef(ctx, rs), rs, nil
}
