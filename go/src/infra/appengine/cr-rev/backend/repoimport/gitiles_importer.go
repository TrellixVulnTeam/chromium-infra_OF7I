// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package repoimport

import (
	"context"
	"fmt"
	"sync"

	"infra/appengine/cr-rev/backend/gitiles"
	"infra/appengine/cr-rev/common"
	"infra/appengine/cr-rev/config"
	"infra/appengine/cr-rev/models"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/logging"
	gitilesProto "go.chromium.org/luci/common/proto/gitiles"
)

const (
	gitilesLogPageSize = 1000
)

type gitilesImporter struct {
	gitiesClient gitiles.Client
	leaser       *leaser
	repo         common.GitRepository
	config       *config.Repository
	// importedBranches keys are git refs
	importedRefs map[string]struct{}
	// importedCommits keys are git commit hashes
	importedCommits stringset.Set
}

// NewGitilesImporter initializes gitilesImporter struct and returns its
// pointer.
func NewGitilesImporter(ctx context.Context, repo common.GitRepository) Importer {
	return &gitilesImporter{
		gitiesClient:    gitiles.GetClient(ctx),
		leaser:          newLeaser(repo),
		repo:            repo,
		importedRefs:    make(map[string]struct{}),
		importedCommits: stringset.New(0),
	}
}

// Run starts import of desired repository. It first acquires a lease which is
// stored in datastore. If successful, it periodically refreshes the lease
// (repoLockUpdateDuration). If the datastore document has changed in any way
// (another process, manually), the lease is considered no longer valid and
// error will be returned.
// Importer will import all desired refs, as defined in repo.config.Refs.
func (imp *gitilesImporter) Run(ctx context.Context) error {
	logging.Debugf(ctx, "running import for: %s/%s", imp.repo.Host, imp.repo.Name)
	refsPaths := []string{common.DefaultIncludeRefs}
	if imp.repo.Config != nil && len(imp.repo.Config.Refs) > 0 {
		refsPaths = imp.repo.Config.Refs
	}

	return imp.leaser.WithLease(ctx, func(ctx context.Context) error {
		return imp.scanRefsPaths(ctx, refsPaths)
	})
}

// scanRefsPaths returns a list of revisions of all references that match
// refsPaths.
func (imp *gitilesImporter) scanRefsPaths(ctx context.Context, refsPaths []string) error {
	refsToScan := []string{}
	for _, refsPath := range refsPaths {
		in := &gitilesProto.RefsRequest{
			Project:  imp.repo.Name,
			RefsPath: refsPath,
		}

		out, err := imp.gitiesClient.Refs(ctx, in)
		if err != nil {
			return err
		}
		logging.Infof(ctx, "Found %d branches in %s/%s",
			len(out.GetRevisions()), imp.repo.Host, imp.repo.Name)
		for ref, rev := range out.GetRevisions() {
			if rev == "" {
				logging.Warningf(ctx, "Reference %s points to empty commit in %s/%s", ref, imp.repo.Host, imp.repo.Name)
				continue
			}
			refsToScan = append(refsToScan, ref)
		}
	}

	logging.Debugf(ctx, "Collected %d revisions from branches in %s/%s",
		len(refsToScan), imp.repo.Host, imp.repo.Name)
	for _, rev := range refsToScan {
		err := imp.traverseRev(ctx, rev)
		if err != nil {
			return err
		}
	}
	return nil
}

// traverseRev traverses desired revision and persists commits to Datastore.
// All visited revisions by this importer are stored in-memory*. If revisions is
// found in memory while traversing, it means some previous traversal has seen
// this path. In that case, we can stop traversal.
//
// * Memory requirements:
// Let's assume 150B are used to store commit information (we don't convert hex
// string -> byte, and we assume hex is always 40 bytes long). To import
// chromium/src, which has close to 1M entries, we need 150MB of memory.
func (imp *gitilesImporter) traverseRev(ctx context.Context, ref string) error {
	if !imp.repo.ShouldIndex(ref) {
		logging.Infof(ctx, "Skipping scanning ref %s of %s/%s/ (should not index)",
			ref, imp.repo.Host, imp.repo.Name)
		return nil
	}
	logging.Debugf(ctx, "Scanning ref %s of %s/%s/",
		ref, imp.repo.Host, imp.repo.Name)
	in := &gitilesProto.LogRequest{
		Project:    imp.repo.Name,
		Committish: ref,
		PageSize:   gitilesLogPageSize,
	}
	wg := sync.WaitGroup{}

	for done := false; !done; {
		resp, err := imp.gitiesClient.Log(ctx, in)
		if err != nil {
			return fmt.Errorf("error querying Gitiles: %w", err)
		}

		logging.Debugf(ctx, "Found %d commits in %s/%s, ref: %s",
			len(resp.GetLog()), imp.repo.Host, imp.repo.Name, ref)

		commits := []*common.GitCommit{}
		for _, log := range resp.GetLog() {
			commit := common.GitCommit{
				Repository:    imp.repo,
				CommitMessage: log.GetMessage(),
				Hash:          log.GetId(),
			}
			if imp.importedCommits.Has(commit.ID()) {
				// We already imported this commit, and
				// therefore all parent commits.
				done = true
				break
			}
			commits = append(commits, &commit)
			imp.importedCommits.Add(commit.ID())
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			models.PersistCommits(ctx, commits)
		}()

		if resp.GetNextPageToken() == "" {
			done = true
		} else {
			in.PageToken = resp.GetNextPageToken()
		}
	}
	wg.Wait()
	return nil
}
