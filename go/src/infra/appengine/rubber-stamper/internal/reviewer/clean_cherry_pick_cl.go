// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reviewer

import (
	"context"
	"fmt"
	"strings"

	gerritpb "go.chromium.org/luci/common/proto/gerrit"

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

	// Check there's only one revision uploaded.
	if t.RevisionsCount > 1 {
		return "The change cannot be reviewed. There are more than one revision uploaded.", nil
	}

	// Check whether the change that this change was cherry-picked from is
	// properly reviewed.
	originalClInfo, err := gc.GetChange(ctx, &gerritpb.GetChangeRequest{
		Number:  t.CherryPickOfChange,
		Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
	})
	if err != nil {
		return "", fmt.Errorf("gerrit GetChange rpc call failed with error: %v", err)
	}

	// Check whether the change is in a configured time window.
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

	// Check whether the change is mergable.
	mi, err := gc.GetMergeable(ctx, &gerritpb.GetMergeableRequest{
		Number:     t.Number,
		Project:    t.Repo,
		RevisionId: t.Revision,
	})
	if err != nil {
		return "", fmt.Errorf("gerrit GetMergeable rpc call failed with error: %v", err)
	}
	if !mi.Mergeable {
		return "The change is not mergeable.", nil
	}
	return "", nil
}