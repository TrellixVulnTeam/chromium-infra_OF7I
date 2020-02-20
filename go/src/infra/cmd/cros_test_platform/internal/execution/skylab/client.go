// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package skylab

import (
	"context"
	"infra/cmd/cros_test_platform/internal/execution/isolate"
	"infra/cmd/cros_test_platform/internal/execution/swarming"
	"infra/libs/skylab/request"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// Client bundles local interfaces to various remote services used by Runner.
type Client struct {
	Swarming      swarming.Client
	IsolateGetter isolate.GetterFactory
}

// ValidateArgs checks whether this test has dependencies satisfied by
// at least one Skylab bot.
func (c *Client) ValidateArgs(ctx context.Context, args *request.Args) (bool, error) {
	dims, err := args.StaticDimensions()
	if err != nil {
		return false, errors.Annotate(err, "validate dependencies").Err()
	}
	exists, err := c.Swarming.BotExists(ctx, dims)
	if err != nil {
		return false, errors.Annotate(err, "validate dependencies").Err()
	}
	if !exists {
		logging.Warningf(ctx, "Dependency validation failed for %s: no bot exists with dimensions %+v.", args.Cmd.TaskName, dims)
	}
	return exists, nil
}
