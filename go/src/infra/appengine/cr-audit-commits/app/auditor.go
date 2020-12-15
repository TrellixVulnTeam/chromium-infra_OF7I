// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/api/gitiles"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/proto/git"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
	ds "go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server/router"

	"infra/appengine/cr-audit-commits/app/config"
	"infra/appengine/cr-audit-commits/app/rules"
)

// Tests can put mock clients here, prod code will ignore this global.
var auditorTestClients *rules.Clients
var configGet = config.GetUpdatedRuleMap

// pauseRefErr is used to indicate that a ref needs to be paused.
var errPauseRef = errors.New("ref needs to be paused")

// taskAuditor is the main entry point for scanning the commits on a given ref
// and auditing those that are relevant to the configuration.
//
// It scans the ref in the repo and creates entries for relevant commits,
// executes the audit functions on such commits, and calls notification
// functions when appropriate.
//
// This is expected to run every 5 minutes and for that reason, it has designed
// to stop 4 minutes 30 seconds and save any partial progress.
func taskAuditor(rc *router.Context) {
	outerCtx, resp := rc.Context, rc.Writer
	// Create a derived context with a 9:30 timeout s.t. we have enough
	// time to save results for at least some of the audited commits,
	// considering that a run of this handler will be scheduled every 10
	// minutes.
	ctx, cancel := context.WithTimeout(outerCtx, 9*time.Minute+30*time.Second)
	defer cancel()

	refURL := rc.Request.FormValue("refUrl")
	cfg, repoState, err := loadConfig(ctx, refURL)
	if err != nil {
		logging.WithError(err).Errorf(ctx, "Failed to load config for %s", refURL)
		http.Error(resp, "", http.StatusBadRequest)
		return
	}

	var cs *rules.Clients
	if auditorTestClients != nil {
		cs = auditorTestClients
	} else {
		httpClient, err := rules.GetAuthenticatedHTTPClient(ctx, rules.GerritScope, rules.EmailScope)
		if err != nil {
			logging.WithError(err).Errorf(ctx, "Failed to get auth client")
			http.Error(resp, "", http.StatusInternalServerError)
			return
		}

		cs, err = rules.InitializeClients(ctx, cfg, httpClient)
		if err != nil {
			logging.WithError(err).Errorf(ctx, "Failed to get initialize clients")
			http.Error(resp, "", http.StatusInternalServerError)
			return
		}
	}

	// Pause the ref if it stops auditing for a long time.
	if !repoState.Paused && time.Now().Sub(repoState.LastUpdatedTime) > config.StuckScannerDuration {
		if err = pauseRefAuditing(ctx, cfg, repoState, cs); err != nil {
			http.Error(resp, "", http.StatusBadGateway)
			return
		}
		logging.Warningf(ctx, "Ref %s is now paused", refURL)
		http.Error(resp, "", http.StatusConflict)
		return
	}

	// If the ref is paused, check if we can unpause it.
	if repoState.Paused {
		logging.Debugf(ctx, "Ref %s is paused: cfg is %v", refURL, cfg)
		if cfg.OverwriteLastKnownCommit != "" && repoState.AcceptedOverwriteLastKnownCommit != cfg.OverwriteLastKnownCommit {
			repoState.AcceptedOverwriteLastKnownCommit = cfg.OverwriteLastKnownCommit
			repoState.LastKnownCommit = cfg.OverwriteLastKnownCommit
			repoState.Paused = false
		} else {
			logging.Warningf(ctx, "Ref %s is still paused", refURL)
			http.Error(resp, "", http.StatusConflict)
			return
		}
	}

	fl, err := getCommitLog(ctx, cfg, repoState, cs)
	if err == errPauseRef {
		if err := pauseRefAuditing(ctx, cfg, repoState, cs); err != nil {
			http.Error(resp, "", http.StatusBadGateway)
		}
		logging.Warningf(ctx, "Ref %s is late paused", refURL)
		http.Error(resp, "", http.StatusConflict)
		return
	} else if err != nil {
		logging.WithError(err).Errorf(ctx, "Failed to get commit logs")
		http.Error(resp, "", http.StatusBadGateway)
		return
	}

	if repoState.Paused {
		logging.Warningf(ctx, "Ref %s is very late paused", refURL)
		http.Error(resp, "", http.StatusConflict)
		return
	}

	// Iterate over the log, creating relevantCommit entries as appropriate
	// and updating repoState. If the context expires during this process,
	// save the repoState and bail.
	if err = scanCommits(ctx, fl, cfg, repoState); err != nil && err != context.DeadlineExceeded {
		logging.WithError(err).Errorf(ctx, "Could not save new relevant commit")
		http.Error(resp, "", http.StatusServiceUnavailable)
		return
	}
	if err == context.DeadlineExceeded {
		logging.Warningf(ctx, "context has expired, not proceeding with audit")
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
		logging.WithError(err).Errorf(ctx, "performScheduledAudits failed")
		http.Error(resp, "", http.StatusInternalServerError)
		return
	}
	if putErr := saveAuditedCommits(outerCtx, auditedCommits, cfg, repoState); putErr != nil {
		logging.WithError(err).Errorf(ctx, "saveAuditedCommits with %d items failed", len(auditedCommits))
		http.Error(resp, "", http.StatusServiceUnavailable)
		return
	}
	if err == context.DeadlineExceeded {
		logging.Warningf(ctx, "context has expired, not proceeding with notifications")
		return
	}
	if err = notifyAboutViolations(outerCtx, cfg, repoState, cs); err != nil {
		logging.WithError(err).Errorf(ctx, "notifyAboutViolations failed")
		http.Error(resp, "", http.StatusBadGateway)
		return
	}
}

// getCommitLog gets from gitiles a list from the tip to the last known commit
// of the ref in reverse chronological (as per git parentage) order.
func getCommitLog(ctx context.Context, cfg *rules.RefConfig, repoState *rules.RepoState, cs *rules.Clients) ([]*git.Commit, error) {

	// Fetch git log from gitiles with a timeout of 5 minutes.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	host, project, err := gitiles.ParseRepoURL(cfg.BaseRepoURL)
	if err != nil {
		return nil, err
	}

	gc, err := cs.NewGitilesClient(host)
	if err != nil {
		logging.WithError(err).Errorf(ctx, "Could not create new gitiles client for %s", host)
		return nil, err
	}

	// Get the tip of the repo
	branchName := cfg.BranchName
	if !strings.HasPrefix(branchName, "refs/") {
		branchName = "refs/heads/" + branchName
	}
	logging.Debugf(ctx, "branchName: %s", branchName)
	resp, err := gc.Refs(ctx, &gitilespb.RefsRequest{Project: project, RefsPath: branchName})
	if err != nil {
		logging.WithError(err).Errorf(ctx, "Could not get the tip of the ref %s", project)
		return nil, err
	}
	newHead, ok := resp.Revisions[fmt.Sprintf("%s/%s", branchName, branchName)]
	if !ok {
		return nil, fmt.Errorf("Could not get the branch %s in ref %s", branchName, project)
	}
	oldHead := repoState.LastKnownCommit

	logReq := &gitilespb.LogRequest{
		Project:            project,
		ExcludeAncestorsOf: oldHead,
		Committish:         newHead,
	}

	fl, err := gitiles.PagingLog(ctx, gc, logReq, config.MaxCommitsPerRefUpdate)
	switch status.Code(err) {
	case codes.OK:
		// If fetched too many commits, pause auditing.
		if len(fl) >= config.MaxCommitsPerRefUpdate {
			return nil, errPauseRef
		}
		return fl, nil
	case codes.NotFound:
		// Handled below
		break
	default:
		// Gitiles accidental error
		logging.WithError(err).Errorf(ctx, "Could not get children of revision %s from gitiles",
			oldHead)
		return nil, err
	}

	// Either:
	//  (1) oldHead is no longer known in gitiles (force push),
	//  (2) newHead is no longer known in gitiles (eventual consistency,
	//     or concurrent force push executed just now, or ACLs change)
	//  (3) gitiles accidental 404, aka fluke.
	// In case (1), the ref will stop auditing and a bug will be filed for
	// oncalls to handle the problem.
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
		return nil, newErr
	}

	// Fetch log of oldHead only
	_, oldErr := gitiles.PagingLog(ctx, gc, &gitilespb.LogRequest{
		Project:    project,
		Committish: oldHead,
	}, 1)
	switch status.Code(oldErr) {
	case codes.NotFound:
		// case (1)
		return nil, errPauseRef
	case codes.OK:
		return nil,
			fmt.Errorf("Weirdly, log(%s) and log(%s) work, but not log(%s..%s)",
				oldHead, newHead, oldHead, newHead)
	default:
		// case (3)
		logging.WithError(oldErr).Errorf(ctx, "Could not get gitiles log from revision %s", oldHead)
		return nil, oldErr
	}
}

// scanCommits iterates over the list of commits in the given log, decides if
// each is relevant to any of the configured rulesets and creates records for
// each that is. Also updates the record for the ref, but does not persist it,
// this is instead done by Auditor after this function is executed. This is left
// to the handler in case the context given to this function expires before
// reaching the end of the log.
func scanCommits(ctx context.Context, fl []*git.Commit, cfg *rules.RefConfig, repoState *rules.RepoState) error {
	// Prepare all the relevant commits.
	var relevantCommits []*git.Commit
	for _, commit := range fl {
		for _, ruleSet := range cfg.Rules {
			if ruleSet.MatchesCommit(commit) {
				relevantCommits = append(relevantCommits, commit)
				break
			}
		}
	}

	// Split commits in batches and transactionally save each batch in
	// datastore.
	const batchSize = 100
	for i := len(relevantCommits); i > 0; i -= batchSize {
		batchBegin := i - batchSize
		if batchBegin < 0 {
			batchBegin = 0
		}

		prc := repoState.LastRelevantCommit
		if i < len(relevantCommits) {
			prc = relevantCommits[i].Id
		}

		err := saveNewRelevantCommits(ctx, repoState, relevantCommits[batchBegin:i], prc)
		if err != nil {
			logging.WithError(err).Errorf(ctx, "Failed to save relevant commits in ref %s",
				cfg.RepoURL())
			return err
		}
	}

	// Update repoState.
	if len(fl) > 0 {
		repoState.LastKnownCommit = fl[0].Id
		// Ignore possible error, this time is used for display purposes only.
		if fl[0].Committer != nil {
			ct, _ := ptypes.Timestamp(fl[0].Committer.Time)
			repoState.LastKnownCommitTime = ct
		}
	}
	if len(relevantCommits) > 0 {
		repoState.LastRelevantCommit = relevantCommits[0].Id
		if relevantCommits[0].Committer != nil {
			ct, _ := ptypes.Timestamp(relevantCommits[0].Committer.Time)
			repoState.LastRelevantCommitTime = ct
		}
	}
	repoState.LastUpdatedTime = time.Now().UTC()

	if err := ds.Put(ctx, repoState); err != nil {
		logging.WithError(err).Errorf(ctx, "Could not save repoState %s", cfg.RepoURL())
		return err
	}

	return nil
}

func saveNewRelevantCommits(ctx context.Context, state *rules.RepoState, commits []*git.Commit, previousRelevantCommit string) error {
	rk := ds.KeyForObj(ctx, state)

	rcs := make([]*rules.RelevantCommit, len(commits))
	for i, commit := range commits {
		commitTime, err := ptypes.Timestamp(commit.GetCommitter().GetTime())
		if err != nil {
			logging.WithError(err).Errorf(ctx, "Invalid commit time: %s", commitTime)
			return err
		}

		prc := previousRelevantCommit
		if i < len(commits)-1 {
			prc = commits[i+1].Id
		}

		rcs[i] = &rules.RelevantCommit{
			RepoStateKey:           rk,
			CommitHash:             commit.Id,
			PreviousRelevantCommit: prc,
			Status:                 rules.AuditScheduled,
			CommitTime:             commitTime,
			CommitterAccount:       commit.Committer.Email,
			AuthorAccount:          commit.Author.Email,
			CommitMessage:          commit.Message,
		}
	}

	exists, err := ds.Exists(ctx, rcs)
	if err != nil {
		logging.WithError(err).Errorf(ctx, "Failed to check existence of commits in ref %s",
			state.ConfigName)
		return err
	}

	putLength := 0
	for i, rc := range rcs {
		if !exists.List(0)[i] {
			rcs[putLength] = rc
			putLength++
		}
	}

	if putLength == 0 {
		return nil
	}

	if err := ds.Put(ctx, rcs[:putLength]); err != nil {
		logging.WithError(err).Errorf(ctx, "Could not save commits from %s to %s in ref %s",
			rcs[0].CommitHash, rcs[putLength-1].CommitHash, state.ConfigName)
		return err
	}
	logging.Infof(ctx, "Saved commits from %s to %s in ref %s",
		rcs[0].CommitHash, rcs[putLength-1].CommitHash, state.ConfigName)
	return nil
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
				AuditedCommits.Add(ctx, 1, c.Status.ToShortString(), cfg.RepoURL())
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

// pauseRefAuditing pauses auditing of a ref and file a bug for it.
func pauseRefAuditing(ctx context.Context, cfg *rules.RefConfig, repoState *rules.RepoState, cs *rules.Clients) error {
	return ds.RunInTransaction(ctx, func(ctx context.Context) error {
		err := reportRefFailure(ctx, cfg, repoState, cs)
		if err != nil {
			logging.WithError(err).Errorf(ctx,
				"Could not file a bug for ref %s which stops auditing", cfg.RepoURL())
			return err
		}

		logging.Infof(ctx, "Pausing ref %s", cfg.RepoURL())
		repoState.Paused = true
		err = ds.Put(ctx, repoState)
		if err != nil {
			logging.WithError(err).Errorf(ctx, "Could not save repoState for ref %s", cfg.RepoURL())
			return err
		}
		return nil
	}, nil)
}

// reportRefFailure is meant to file a bug when a ref stops auditing for a long
// time or when a non-fast-forwarded update happens.
func reportRefFailure(ctx context.Context, cfg *rules.RefConfig, repoState *rules.RepoState, cs *rules.Clients) error {
	summary := fmt.Sprintf("Failed to get commits from %s", repoState.ConfigName)
	description := fmt.Sprintf(`Failed to get commit from %s. This could possibly be caused by:
1) A non-fast-forward update makes LastKnownCommit become an unaccessible commit;
2) Gitiles git.Log API returns too many commits or encounters some errors, which
causes the time diff between current time and LastUpdatedTime to exceed staleHours.

LastUpdatedTime: %s
LastKnownCommit: %s`, cfg.RepoURL(), repoState.LastUpdatedTime,
		cfg.LinkToCommit(repoState.LastKnownCommit))

	_, err := rules.PostIssue(ctx, cfg, summary, description, cs,
		[]string{"Infra>Security>Audit"}, []string{"GetRefCommitFailure"})
	return err
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

	cfg, ok := configGet(ctx)[rs.ConfigName]
	if !ok {
		return nil, nil, fmt.Errorf("Unknown or missing config %s", rs.ConfigName)
	}
	return cfg.SetConcreteRef(ctx, rs), rs, nil
}
