// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package upload

import (
	"context"

	"go.chromium.org/luci/common/errors"
)

// Params is a collection of parameters for uploading a directory.
type Params struct {
	// Dir is the path to the primary directory to offload.
	//
	// Example: /path/to/swarming/logs
	Dir string
	// GSURL is the destination path.
	GSURL string
}

// Upload a list of directories to Google Storage.
func Upload(ctx context.Context, params *Params) error {
	return errors.Reason("upload: not yet implemented").Err()
}
