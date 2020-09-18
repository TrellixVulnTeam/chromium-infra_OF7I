// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pubsub

import (
	"context"
	"errors"
	"fmt"
	"infra/appengine/cr-rev/backend/gitiles"
	"infra/appengine/cr-rev/common"
	"infra/appengine/cr-rev/config"
	"infra/appengine/cr-rev/models"
	"strings"

	"go.chromium.org/luci/common/logging"
	gitilesProto "go.chromium.org/luci/common/proto/gitiles"
)

// Processor returns function which is called each time there is a new
// SourceRepoEvent.
func Processor(host *config.Host) ProcessPubsubMessage {
	reposConfig := map[string]*config.Repository{}
	for _, repo := range host.GetRepos() {
		reposConfig[repo.GetName()] = repo
	}

	return func(ctx context.Context, m *SourceRepoEvent) error {
		chunks := strings.SplitN(m.GetName(), "/", 4)
		if len(chunks) != 4 {
			logging.Errorf(ctx, "Invalid name format. Host: %s, name: %s", host.GetName(), m.GetName())
			return errors.New("Invalid repository format")
		}
		repository := common.GitRepository{
			Host:   host.GetName(),
			Name:   chunks[3],
			Config: reposConfig[chunks[3]],
		}

		events := m.GetRefUpdateEvent()
		if events == nil {
			return nil
		}
		for _, event := range events.GetRefUpdates() {
			ref := event.GetRefName()

			if !repository.ShouldIndex(ref) {
				logging.Debugf(ctx, "Skipping indexing %v on ref %s", repository, ref)
				continue
			}
			err := importCommits(ctx, repository, event.GetOldId(), event.GetNewId())
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func importCommits(ctx context.Context, repository common.GitRepository, from, to string) error {
	c := gitiles.GetClient(ctx)
	req := &gitilesProto.LogRequest{
		Project:            repository.Name,
		Committish:         to,
		ExcludeAncestorsOf: from,
		PageSize:           1000,
	}
	for {
		resp, err := c.Log(ctx, req)
		if err != nil {
			return fmt.Errorf("error querying Gitiles: %w", err)
		}
		logging.Debugf(ctx, "Found %d commits in %v, %s..%s",
			len(resp.GetLog()), repository, from, to)

		commits := []*common.GitCommit{}
		for _, log := range resp.GetLog() {
			commit := &common.GitCommit{
				Repository:    repository,
				CommitMessage: log.GetMessage(),
				Hash:          log.GetId(),
			}
			commits = append(commits, commit)
		}
		err = models.PersistCommits(ctx, commits)
		if err != nil {
			return fmt.Errorf("error persisting data: %w", err)
		}
		if resp.GetNextPageToken() == "" {
			return nil
		}
		req.PageToken = resp.GetNextPageToken()
	}
}
