// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reviewer

import (
	"context"
	"path"
	"sort"

	"go.chromium.org/luci/common/logging"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"

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

	if hostCfg == nil || hostCfg.BenignFilePattern == nil {
		invalidFiles := make([]string, 0, len(resp.Files))
		for file := range resp.Files {
			invalidFiles = append(invalidFiles, file)
		}
		return invalidFiles, nil
	}

	fileExtensionMap := hostCfg.BenignFilePattern.FileExtensionMap

	var invalidFiles []string
	for file := range resp.Files {
		isValid := false
		ext := path.Ext(file)
		if _, ok := fileExtensionMap[ext]; !ok {
			invalidFiles = append(invalidFiles, file)
			continue
		}
		for _, p := range fileExtensionMap[ext].Paths {
			ok, err := path.Match(p, file)
			if err != nil {
				logging.WithError(err).Errorf(ctx, "invalid path in BenignFilePattern: %s", p)
				continue
			}
			if ok {
				isValid = true
				break
			}
		}
		if !isValid {
			invalidFiles = append(invalidFiles, file)
		}
	}

	sort.Strings(invalidFiles)
	return invalidFiles, nil
}
