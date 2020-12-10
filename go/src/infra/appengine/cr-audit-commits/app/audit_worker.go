// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"infra/appengine/cr-audit-commits/app/rules"

	"go.chromium.org/luci/common/logging"
	ds "go.chromium.org/luci/gae/service/datastore"
)

const (
	// MaxWorkers is the upper limit of how many worker goroutines to spawn.
	// There's nothing special about 16, but it seems like a reasonable
	// number of goroutines to share a cpu while waiting for i/o.
	MaxWorkers = 16
	// CommitsPerWorker is 10 commits per goroutine. To keep the life of the
	// cron job short. It is unlikely that we'll ever have to audit more
	// than this many commits in a single run of the cron job.
	CommitsPerWorker = 10
)

// workerParams are passed on to the workers for communication.
type workerParams struct {
	// Immutable.
	ap      rules.AuditParams
	rules   map[string]rules.AccountRules
	clients *rules.Clients

	// jobs is the commits to work on.
	jobs chan *rules.RelevantCommit

	// audited is the results.
	audited chan *rules.RelevantCommit
}

// performScheduledAudits queries the datastore for commits that need to be
// audited, spawns a pool of goroutines and sends jobs to perform each audit
// to the pool. When it's done, it returns a map of commit hash to the commit
// entity.
//
// If the context expires while auditing, this function will return the partial
// results along with the appropriate error for the caller to handle persisting
// the partial results and thus avoid duplicating work.
func performScheduledAudits(ctx context.Context, cfg *rules.RefConfig, repoState *rules.RepoState, cs *rules.Clients) (map[string]*rules.RelevantCommit, error) {
	cfgk := ds.KeyForObj(ctx, repoState)
	pcq := ds.NewQuery("RelevantCommit").Ancestor(cfgk).Eq("Status", rules.AuditPending).Limit(MaxWorkers * CommitsPerWorker)
	ncq := ds.NewQuery("RelevantCommit").Ancestor(cfgk).Eq("Status", rules.AuditScheduled).Limit(MaxWorkers * CommitsPerWorker)

	// Count the number of commits to be analyzed to estimate a reasonable
	// number of workers for the load.
	nNewCommits, err := ds.Count(ctx, ncq)
	if err != nil {
		return nil, err
	}
	nPendingCommits, err := ds.Count(ctx, pcq)
	if err != nil {
		return nil, err
	}
	nCommits := nNewCommits + nPendingCommits
	if nCommits == 0 {
		logging.Debugf(ctx, "No relevant commits to audit")
		return nil, nil
	}
	logging.Debugf(ctx, "Auditing %d commits", nCommits)

	// Make the number of workers proportional to the number of commits
	// that need auditing.
	nWorkers := 1 + int(nCommits)/2
	// But make sure they don't exceed a certain limit.
	if nWorkers > MaxWorkers {
		nWorkers = MaxWorkers
	}

	wp := &workerParams{
		ap: rules.AuditParams{
			RepoCfg:   cfg,
			RepoState: repoState,
		},
		rules:   cfg.Rules,
		clients: cs,
		jobs:    make(chan *rules.RelevantCommit, nWorkers*CommitsPerWorker),
		audited: make(chan *rules.RelevantCommit),
	}
	wg := sync.WaitGroup{}
	for i := 0; i < nWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wp.auditWorker(ctx)
		}()
	}
	go func() {
		wg.Wait()
		close(wp.audited)
	}()

	go func() {
		defer close(wp.jobs)
		// Send pending audit jobs to workers.
		ds.Run(ctx, pcq, func(rc *rules.RelevantCommit) {
			wp.jobs <- rc
		})
		// Send scheduled audit jobs to workers.
		ds.Run(ctx, ncq, func(rc *rules.RelevantCommit) {
			wp.jobs <- rc
		})
	}()

	auditedCommits := map[string]*rules.RelevantCommit{}
	for auditedCommit := range wp.audited {
		auditedCommits[auditedCommit.CommitHash] = auditedCommit
	}
	return auditedCommits, ctx.Err()
}

// auditWorker is the main goroutine for each worker.
func (wp *workerParams) auditWorker(ctx context.Context) {
	for rc := range wp.jobs {
		select {
		case <-ctx.Done():
			return
		default:
			start := time.Now()
			runRules(ctx, rc, wp.rules, wp.ap, wp.clients)
			wp.audited <- rc
			PerCommitAuditDuration.Add(ctx, time.Since(start).Seconds()*1000.0, rc.Status.ToShortString(), wp.ap.RepoState.ConfigName)
		}
	}
}

// runRules runs rules for one commit, aggregate the results and write it to
// the audited channel.
//
// It swallows any error, only logging it in order to move to the next commit.
func runRules(ctx context.Context, rc *rules.RelevantCommit, rm map[string]rules.AccountRules, ap rules.AuditParams, clients *rules.Clients) {
	for _, rs := range rm {
		// TODO(xinyuoffline): Only retry transient errors.
		// TODO(xinyuoffline): Alert on permanent errors.
		// TODO(xinyuoffline): https://cloud.google.com/error-reporting/docs/formatting-error-messages ?
		if err := runAccountRules(ctx, rs, rc, ap, clients); err != nil {
			rc.Retries++
			logging.WithError(err).Errorf(ctx, "audit failed")
			if err != context.DeadlineExceeded {
				logging.Warningf(ctx, "Discarding incomplete results: %s", rc.Result)
				rc.Result = []rules.RuleResult{}
				if rc.Retries > rules.MaxRetriesPerCommit {
					rc.Status = rules.AuditFailed
				}
			}
			return
		}
	}

	if rc.Status == rules.AuditScheduled || rc.Status == rules.AuditPending { // No rules failed.
		rc.Status = rules.AuditCompleted
		// If any rules are pending to be decided, leave the commit as pending.
		for _, rr := range rc.Result {
			if rr.RuleResultStatus == rules.RulePending {
				rc.Status = rules.AuditPending
				break
			}
		}
	}
}

// runAccountRules runs each rules in an AccountRules on a commit.
func runAccountRules(ctx context.Context, rs rules.AccountRules, rc *rules.RelevantCommit, ap rules.AuditParams, clients *rules.Clients) error {
	if !rs.MatchesRelevantCommit(rc) {
		return nil
	}
	ap.TriggeringAccount = rc.AuthorAccount
	if rs.Account != "*" {
		ap.TriggeringAccount = rs.Account
	}
	for _, r := range rs.Rules {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			previousResult := rules.PreviousResult(ctx, rc, r.GetName())
			if previousResult == nil || previousResult.RuleResultStatus == rules.RulePending {
				pCurrentRuleResult, err := r.Run(ctx, &ap, rc, clients)
				if err != nil {
					return fmt.Errorf(
						"%s: %w\nCommit: %s/+/%s\nBranch: %s\nAuthor: %s\nSubject: %q", r.GetName(), err,
						ap.RepoCfg.BaseRepoURL, rc.CommitHash, ap.RepoCfg.BranchName, rc.AuthorAccount, rc.CommitMessage)
				}

				currentRuleResult := *pCurrentRuleResult
				currentRuleResult.RuleName = r.GetName()
				updated := rc.SetResult(currentRuleResult)
				if updated && (currentRuleResult.RuleResultStatus == rules.RuleFailed || currentRuleResult.RuleResultStatus == rules.NotificationRequired) {
					rc.Status = rules.AuditCompletedWithActionRequired
				}
			}
		}
	}
	return nil
}
