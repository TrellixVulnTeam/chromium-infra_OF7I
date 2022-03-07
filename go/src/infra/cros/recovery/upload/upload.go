// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package upload

import (
	"context"

	"go.chromium.org/luci/common/errors"
	lucigs "go.chromium.org/luci/common/gcloud/gs"

	"infra/libs/skylab/gs"
)

// Params is a collection of parameters for uploading a directory.
type Params struct {
	// Dir is the path to the primary directory to offload.
	//
	// Example: /path/to/swarming/logs
	SourceDir string
	// GSURL is the destination path.
	GSURL string
	// MaxConcurrentJobs is the maximum number of concurrent uploads that can happen at once
	MaxConcurrentJobs int
}

// GSClient is a Google Storage client.
//
// This interface is a subset of the gs.Client interface.
type gsClient interface {
	// NewWriter creates a new google storage writer rooted at a gs path.
	NewWriter(p lucigs.Path) (lucigs.Writer, error)
}

// Upload a list of directories to Google Storage.
func Upload(ctx context.Context, gsClient gsClient, params *Params) error {
	if gsClient == nil {
		return errors.Reason("upload: client cannot be nil").Err()
	}
	if params == nil {
		return errors.Reason("upload: params cannot be nil").Err()
	}
	if params.MaxConcurrentJobs <= 0 {
		return errors.Reason("upload: max jobs must be positive").Err()
	}
	dirWriter := gs.NewDirWriter(gsClient, params.MaxConcurrentJobs)
	if err := dirWriter.WriteDir(ctx, params.SourceDir, lucigs.Path(params.GSURL)); err != nil {
		return errors.Annotate(err, "upload: upload main directory").Err()
	}
	return nil
}
