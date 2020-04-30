// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"time"

	"context"

	ds "go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/logging"

	"infra/appengine/cr-audit-commits/app/rules"
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
	// To send tasks to the worker goroutines.
	jobs chan *rules.RelevantCommit

	// To receive results from the worker goroutines.
	audited chan *rules.RelevantCommit

	// Every worker is guaranteed to signal this channel.
	workerFinished chan bool

	// But only those that finish cleanly will signal this one.
	finishedCleanly chan bool

	// These read-only globals are meant to be read by the goroutines.
	rules map[string]rules.AccountRules

	clients *rules.Clients
}

// performScheduledAudits queries the datastore for commits that need to be
// audited, spawns a pool of goroutines and sends jobs to perform each audit
// to the pool. When it's done, it returns a map of commit hash to the commit
// entity.
//
// if the context expires while auditing, this function will return the partial
// results along with the appropriate error for the caller to handle persisting
// the partial results and thus avoid duplicating work.
func performScheduledAudits(ctx context.Context, cfg *rules.RefConfig, refState *rules.RefState, cs *rules.Clients) (map[string]*rules.RelevantCommit, error) {
	auditedCommits := make(map[string]*rules.RelevantCommit)
	cfgk := ds.KeyForObj(ctx, refState)

	ap := rules.AuditParams{
		RepoCfg:  cfg,
		RefState: refState,
	}

	pcq := ds.NewQuery("RelevantCommit").Ancestor(cfgk).Eq("Status", rules.AuditPending).Limit(MaxWorkers * CommitsPerWorker)
	ncq := ds.NewQuery("RelevantCommit").Ancestor(cfgk).Eq("Status", rules.AuditScheduled).Limit(MaxWorkers * CommitsPerWorker)

	wp := &workerParams{rules: cfg.Rules, clients: cs}

	// Count the number of commits to be analyzed to estimate a reasonable
	// number of workers for the load.
	nNewCommits, err := ds.Count(ctx, ncq)
	nPendingCommits, err := ds.Count(ctx, pcq)
	nCommits := nNewCommits + nPendingCommits
	if err != nil {
		return auditedCommits, err
	}
	if nCommits == 0 {
		logging.Infof(ctx, "No relevant commits to audit")
		return auditedCommits, nil
	}
	logging.Infof(ctx, "Auditing %d commits", nCommits)

	// Make the number of workers proportional to the number of commits
	// that need auditing.
	nWorkers := 1 + int(nCommits)/2
	// But make sure they don't exceed a certain limit.
	if nWorkers > MaxWorkers {
		nWorkers = MaxWorkers
	}

	logging.Infof(ctx, "Starting %d workers", nWorkers)
	wp.jobs = make(chan *rules.RelevantCommit, nWorkers*CommitsPerWorker)
	wp.audited = make(chan *rules.RelevantCommit, nWorkers*CommitsPerWorker)
	wp.workerFinished = make(chan bool, nWorkers)
	wp.finishedCleanly = make(chan bool, nWorkers)
	for i := 0; i < nWorkers; i++ {
		i := i
		go audit(ctx, i, ap, wp, refState.ConfigName)
	}

	// Send pending audit jobs to workers.
	ds.Run(ctx, pcq, func(rc *rules.RelevantCommit) {
		logging.Infof(ctx, "Sending %s to worker pool", rc.CommitHash)
		wp.jobs <- rc
	})
	// Send scheduled audit jobs to workers.
	ds.Run(ctx, ncq, func(rc *rules.RelevantCommit) {
		logging.Infof(ctx, "Sending %s to worker pool", rc.CommitHash)
		wp.jobs <- rc
	})
	// Signal that no more jobs will be sent.
	close(wp.jobs)
	// Wait for all workers to finish.
	for i := 0; i < nWorkers; i++ {
		<-wp.workerFinished
	}
	// Read results into a map.
	close(wp.audited)
	for auditedCommit := range wp.audited {
		auditedCommits[auditedCommit.CommitHash] = auditedCommit
	}

	select {
	case <-ctx.Done():
		// If the context expired, let the caller know by passing this.
		return auditedCommits, context.DeadlineExceeded
	default:
		return auditedCommits, nil
	}
}

// This is the main goroutine for each worker.
func audit(ctx context.Context, n int, ap rules.AuditParams, wp *workerParams, repo string) {
	defer func() { wp.workerFinished <- true }()
	for job := range wp.jobs {
		select {
		case <-ctx.Done():
			return
		default:
			logging.Infof(ctx, "Worker %d about to run job %s", n, job.CommitHash)
			start := time.Now()
			runRules(ctx, job, ap, wp)
			PerCommitAuditDuration.Add(ctx, time.Now().Sub(start).Seconds()*1000.0, job.Status.ToShortString(), repo)
		}
	}
	logging.Infof(ctx, "Worker %d sees no more jobs in the channel", n)
	wp.finishedCleanly <- true
}

// The worker goroutine call this for each commit.
//
// It will run each rule on the commit, aggregate the results save them to the
// datastore entity and finally write it to the audited channel for a
// transaction to persist it when all workers are done.
//
// It swallows any error, only logging it in order to move to the next commit.
func runRules(ctx context.Context, rc *rules.RelevantCommit, ap rules.AuditParams, wp *workerParams) {
	for _, rs := range wp.rules {
		hasExpired, err := runAccountRules(ctx, rs, rc, ap, wp)
		if hasExpired {
			return
		}
		if err != nil {
			rc.Retries++
			logging.Errorf(ctx,
				"Some rule had an error while auditing %s with message: %s", rc.CommitHash, err)
			logging.Warningf(ctx, "Discarding incomplete results: %s", rc.Result)
			rc.Result = []rules.RuleResult{}
			if rc.Retries > rules.MaxRetriesPerCommit {
				rc.Status = rules.AuditFailed
			}
			// Send through the channel anyway to persist the retry
			// counter, and possibly change of status.
			wp.audited <- rc
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
	wp.audited <- rc
}

// Run each rules in an AccountRules on a commit.
//
// It returns a boolean indicating if the context expired, and an error if any
// of the rules' result cannot be decided.
func runAccountRules(ctx context.Context, rs rules.AccountRules, rc *rules.RelevantCommit, ap rules.AuditParams, wp *workerParams) (bool, error) {
	if rs.MatchesRelevantCommit(rc) {
		ap.TriggeringAccount = rc.AuthorAccount
		if rs.Account != "*" {
			ap.TriggeringAccount = rs.Account
		}
		for _, r := range rs.Rules {
			select {
			case <-ctx.Done():
				rc.Retries++
				wp.audited <- rc
				return true, nil
			default:
				previousResult := rules.PreviousResult(ctx, rc, r.GetName())
				if previousResult == nil || previousResult.RuleResultStatus == rules.RulePending {
					pCurrentRuleResult, err := r.Run(ctx, &ap, rc, wp.clients)
					if err != nil {
						return false, err
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
	}
	return false, nil
}
