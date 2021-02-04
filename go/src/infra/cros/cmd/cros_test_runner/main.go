// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner/steps"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/luciexe/build"
	"infra/cros/cmd/cros_test_runner/execute"
)

func main() {
	input := &steps.RunTestsRequest{}
	var writeOutputProps func(*steps.RunTestsResponse)
	var mergeOutputProps func(*steps.RunTestsResponse)

	build.Main(input, &writeOutputProps, &mergeOutputProps,
		func(ctx context.Context, args []string, st *build.State) error {
			logging.Infof(ctx, "have input %v", input)
			execute.Run(ctx, input)
			// actual build code here, build is already Start'd
			// input was parsed from build.Input.Properties
			writeOutputProps(&steps.RunTestsResponse{ErrorSummaryMarkdown: "winning!"})
			return nil // will mark the Build as SUCCESS
		},
	)
}
