// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reviewer

import (
	"context"
	"fmt"
	"sort"
	"strings"

	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	"gopkg.in/src-d/go-git.v4/plumbing/format/gitignore"

	"infra/appengine/rubber-stamper/config"
	"infra/appengine/rubber-stamper/internal/gerrit"
	"infra/appengine/rubber-stamper/tasks/taskspb"
)

// reviewCleanCherryPick checks whether a CL meets the requirements of a clean
// cherry-pick. It returns a string and error, where the string indicates why
// the CL is not a clean cherry-pick. When the string is an empty string and
// error is nil, it means the CL is a clean cherry-pick.
func reviewCleanCherryPick(ctx context.Context, cfg *config.Config, gc gerrit.Client, t *taskspb.ChangeReviewTask) (string, error) {
	hostCfg := cfg.HostConfigs[t.Host]
	var ccpp *config.CleanCherryPickPattern
	if hostCfg.RepoConfigs != nil && hostCfg.RepoConfigs[t.Repo] != nil {
		ccpp = hostCfg.RepoConfigs[t.Repo].CleanCherryPickPattern
	}

	// Check whether the current revision made any file changes compared with
	// the initial revision.
	if t.RevisionsCount > 1 {
		listReq := &gerritpb.ListFilesRequest{
			Number:     t.Number,
			RevisionId: t.Revision,
			Base:       "1",
		}
		resp, err := gc.ListFiles(ctx, listReq)
		if err != nil {
			return "", fmt.Errorf("gerrit ListFiles rpc call failed with error: request %+v, error %v", listReq, err)
		}

		var invalidFiles []string
		for file := range resp.Files {
			if file == "/COMMIT_MSG" {
				continue
			} else {
				invalidFiles = append(invalidFiles, file)
			}
		}

		if len(invalidFiles) > 0 {
			sort.Strings(invalidFiles)
			msg := "The current revision changed the following files compared with the initial revision: " + strings.Join(invalidFiles[:], ", ") + "."
			return msg, nil
		}
	}

	// Check whether the change is in a configured time window.
	getChangeReq := &gerritpb.GetChangeRequest{
		Number:  t.CherryPickOfChange,
		Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
	}
	originalClInfo, err := gc.GetChange(ctx, getChangeReq)
	if err != nil {
		return "", fmt.Errorf("gerrit GetChange rpc call failed with error: request %+v, error %v", getChangeReq, err)
	}
	tw := cfg.DefaultTimeWindow
	if hostCfg.CleanCherryPickTimeWindow != "" {
		tw = hostCfg.CleanCherryPickTimeWindow
	}
	if ccpp != nil && ccpp.TimeWindow != "" {
		tw = ccpp.TimeWindow
	}
	validTime, err := getValidTimeFromTimeWindow(tw)
	if err != nil {
		return "", err
	}
	if originalClInfo.Revisions[originalClInfo.CurrentRevision].Created.AsTime().Before(validTime) {
		return fmt.Sprintf("The change is not in the configured time window. Rubber Stamper is only allowed to review cherry-picks within %s %s.", tw[:len(tw)-1], timeWindowToStr[tw[len(tw)-1:]]), nil
	}

	// Check whether the change was cherry-picked after the original CL has
	// been merged.
	mergeMsg := "The change is not cherry-picked after the original CL has been merged."
	if originalClInfo.Status != gerritpb.ChangeStatus_MERGED {
		return mergeMsg, nil
	}
	if originalClInfo.Revisions[originalClInfo.CurrentRevision].Created.AsTime().After(t.Created.AsTime()) {
		return mergeMsg, nil
	}

	// Check whether the change alters any excluded files.
	if ccpp != nil && len(ccpp.ExcludedPaths) > 0 {
		excludedFiles, err := checkExcludedFiles(ctx, ccpp.ExcludedPaths, gc, t)
		if err != nil {
			return "", err
		}
		if len(excludedFiles) > 0 {
			msg := "The change contains the following files which require a human reviewer: " + strings.Join(excludedFiles[:], ", ") + "."
			return msg, nil
		}
	}

	// Check whether the change is mergeable.
	getMergeableReq := &gerritpb.GetMergeableRequest{
		Number:     t.Number,
		Project:    t.Repo,
		RevisionId: t.Revision,
	}
	mi, err := gc.GetMergeable(ctx, getMergeableReq)
	if err != nil {
		return "", fmt.Errorf("gerrit GetMergeable rpc call failed with error: request %+v, error %v", getMergeableReq, err)
	}
	if !mi.Mergeable {
		return "The change is not mergeable.", nil
	}
	return "", nil
}

// bypassFileCheck tells whether the invalid files check can be bypassed for
// this cherry-pick.
func bypassFileCheck(ctx context.Context, invalidFiles []string, hashtags []string, owner string, fr *config.CleanCherryPickPattern_FileCheckBypassRule) bool {
	if fr == nil || len(fr.IncludedPaths) == 0 || fr.GetHashtag() == "" || len(fr.AllowedOwners) == 0 {
		return false
	}

	// File needs to be in the allow list.
	var patterns []gitignore.Pattern
	for _, path := range fr.IncludedPaths {
		patterns = append(patterns, gitignore.ParsePattern(path, nil))
	}
	matcher := gitignore.NewMatcher(patterns)

	for _, file := range invalidFiles {
		if !matcher.Match(splitPath(file), false) {
			return false
		}
	}

	// CL hashtag.
	tagValid := false
	for _, hashtag := range hashtags {
		if hashtag == fr.GetHashtag() {
			tagValid = true
			break
		}
	}
	if !tagValid {
		return false
	}

	// Allowed CL owner.
	ownerValid := false
	for _, allowedOwner := range fr.AllowedOwners {
		if owner == allowedOwner {
			ownerValid = true
			break
		}
	}
	return ownerValid
}
