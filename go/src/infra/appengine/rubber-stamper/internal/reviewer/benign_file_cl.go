// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reviewer

import (
	"context"
	"path/filepath"
	"sort"
	"strings"

	"go.chromium.org/luci/common/logging"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	"gopkg.in/src-d/go-git.v4/plumbing/format/gitignore"

	"infra/appengine/rubber-stamper/config"
	"infra/appengine/rubber-stamper/internal/gerrit"
	"infra/appengine/rubber-stamper/tasks/taskspb"
)

// reviewBegignFileChange checks whether a CL follows the BenignFilePattern.
// It returns an array of strings and error, where the array provides the paths
// of those files which breaks the pattern. Iff the array is empty and error is
// nil, the CL is a benign CL.
func reviewBenignFileChange(ctx context.Context, hostCfg *config.HostConfig, gc gerrit.Client, t *taskspb.ChangeReviewTask) ([]string, error) {
	listReq := &gerritpb.ListFilesRequest{
		Number:     t.Number,
		RevisionId: t.Revision,
	}
	resp, err := gc.ListFiles(ctx, listReq)
	if err != nil {
		return nil, err
	}

	if hostCfg == nil || hostCfg.RepoConfigs[t.Repo] == nil || hostCfg.RepoConfigs[t.Repo].BenignFilePattern == nil {
		logging.Debugf(ctx, "there's no BenignFilePattern config for host %s, cl %d, revision %s: %v", t.Host, t.Number, t.Revision)
		invalidFiles := make([]string, 0, len(resp.Files))
		for file := range resp.Files {
			if file == "/COMMIT_MSG" {
				continue
			}

			invalidFiles = append(invalidFiles, file)
		}
		return invalidFiles, nil
	}

	var patterns []gitignore.Pattern
	for _, path := range hostCfg.RepoConfigs[t.Repo].BenignFilePattern.Paths {
		patterns = append(patterns, gitignore.ParsePattern(path, nil))
	}
	matcher := gitignore.NewMatcher(patterns)

	var invalidFiles []string
	for file := range resp.Files {
		if file == "/COMMIT_MSG" {
			continue
		}

		if !matcher.Match(splitPath(file), false) {
			invalidFiles = append(invalidFiles, file)
		}
	}

	sort.Strings(invalidFiles)
	return invalidFiles, nil
}

// splitPath splits a path into components, as weird go-git.v4 API wants it.
func splitPath(p string) []string {
	return strings.Split(filepath.Clean(p), string(filepath.Separator))
}
