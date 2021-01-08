// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reviewer

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	"gopkg.in/src-d/go-git.v4/plumbing/format/gitignore"

	"infra/appengine/rubber-stamper/config"
	"infra/appengine/rubber-stamper/internal/gerrit"
	"infra/appengine/rubber-stamper/tasks/taskspb"
)

var (
	timeWindowToDuration = map[string]time.Duration{
		"s": time.Second,
		"m": time.Minute,
		"h": time.Hour,
		"d": 24 * time.Hour,
	}
	timeWindowToStr = map[string]string{
		"s": "second(s)",
		"m": "minute(s)",
		"h": "hour(s)",
		"d": "day(s)",
	}
)

// reviewCleanRevert checks whether a CL meets the requirement of a clean
// revert. It returns a string and error, where the string indicates why the CL
// is not a clean revert. When the string is an empty string and error is nil,
// it means the CL is a clean revert.
func reviewCleanRevert(ctx context.Context, cfg *config.Config, gc gerrit.Client, t *taskspb.ChangeReviewTask) (string, error) {
	hostCfg := cfg.HostConfigs[t.Host]
	var crp *config.CleanRevertPattern
	if hostCfg.RepoConfigs != nil && hostCfg.RepoConfigs[t.Repo] != nil {
		crp = hostCfg.RepoConfigs[t.Repo].CleanRevertPattern
	}

	// Check gerrit GetPureRevert api.
	resp, err := gc.GetPureRevert(ctx, &gerritpb.GetPureRevertRequest{
		Number:  t.Number,
		Project: t.Repo,
	})
	if err != nil {
		return "", fmt.Errorf("failed to call Gerrit GetPureRevert API: %v", err)
	}
	if !resp.IsPureRevert {
		return "Gerrit GetPureRevert API does not mark this CL as a pure revert.", nil
	}

	// Check whether the change is in a configured time window.
	tw := cfg.DefaultTimeWindow
	if hostCfg.CleanRevertTimeWindow != "" {
		tw = hostCfg.CleanRevertTimeWindow
	}
	if crp != nil && crp.TimeWindow != "" {
		tw = crp.TimeWindow
	}
	ok, err := checkTimeWindow(ctx, tw, gc, t)
	if err != nil {
		return "", err
	}
	if !ok {
		return fmt.Sprintf("The change is not in the configured time window. Rubber Stamper is only allowed to review reverts within %s %s.", tw[:len(tw)-1], timeWindowToStr[tw[len(tw)-1:]]), nil
	}

	// Check whether the change alters any excluded files.
	if crp != nil && len(crp.ExcludedPaths) > 0 {
		excludedFiles, err := checkExcludedFiles(ctx, crp.ExcludedPaths, gc, t)
		if err != nil {
			return "", err
		}
		if len(excludedFiles) > 0 {
			msg := "The change contains the following files which require a human reviewer: " + strings.Join(excludedFiles[:], ", ")
			return msg, nil
		}
	}

	// Passed all the checks.
	return "", nil
}

// Check whether the change is inside a valid time window.
func checkTimeWindow(ctx context.Context, tw string, gc gerrit.Client, t *taskspb.ChangeReviewTask) (bool, error) {
	val, err := strconv.Atoi(tw[:len(tw)-1])
	if err != nil || timeWindowToStr[tw[len(tw)-1:]] == "" {
		return false, fmt.Errorf("invalid time_window config %s: %v", tw, err)
	}
	duration := timeWindowToDuration[tw[len(tw)-1:]] * time.Duration(val)
	validTime := time.Now().Add(-duration)

	resp, err := gc.GetChange(ctx, &gerritpb.GetChangeRequest{
		Number:  t.RevertOf,
		Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
	})
	if resp.Revisions[resp.CurrentRevision].Created.AsTime().Before(validTime) {
		return false, nil
	}
	return true, nil
}

// Check whether the change alters any excluded files. Returns a list of
// excluded files and error.
func checkExcludedFiles(ctx context.Context, excludedPaths []string, gc gerrit.Client, t *taskspb.ChangeReviewTask) ([]string, error) {
	listReq := &gerritpb.ListFilesRequest{
		Number:     t.Number,
		RevisionId: t.Revision,
	}
	resp, err := gc.ListFiles(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gerrit ListFiles API: %v", err)
	}

	var patterns []gitignore.Pattern
	for _, path := range excludedPaths {
		patterns = append(patterns, gitignore.ParsePattern(path, nil))
	}
	matcher := gitignore.NewMatcher(patterns)

	var excludedFiles []string
	for file := range resp.Files {
		if file == "/COMMIT_MSG" {
			continue
		}

		if matcher.Match(splitPath(file), false) {
			excludedFiles = append(excludedFiles, file)
		}
	}

	sort.Strings(excludedFiles)
	return excludedFiles, nil
}
