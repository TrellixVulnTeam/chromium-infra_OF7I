// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execute

import (
	"context"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/luciexe/exe"
	"infra/cros/cmd/cros_test_runner/common"
)

// Args contains all the arguments necessary to Run() an execute step.
type Args struct {
	InputPath      string
	OutputPath     string
	SwarmingTaskID string

	Build *bbpb.Build
	Send  exe.BuildSender
}

// Run is the entry point for cros_test_runner.
func Run(ctx context.Context, args Args) error {
	logging.Infof(ctx, "Starting test_runner::execute.")
	var request skylab_test_runner.Request
	if err := common.ReadIntoRequest(args.InputPath, &request); err != nil {
		return err
	}
	if err := validateRequest(request); err != nil {
		return err
	}
	return common.WriteResponse(
		args.OutputPath,
		&skylab_test_runner.Result{},
	)
}

func validateRequest(r skylab_test_runner.Request) error {
	return nil
}
