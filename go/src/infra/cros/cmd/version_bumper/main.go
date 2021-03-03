// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"

	"infra/cros/cmd/version_bumper/execute"

	vpb "go.chromium.org/chromiumos/infra/proto/go/chromiumos/version_bumper"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/luciexe/build"
)

func main() {
	input := &vpb.BumpVersionRequest{}
	var writeOutputProps func(*vpb.BumpVersionResponse)
	var mergeOutputProps func(*vpb.BumpVersionResponse)

	build.Main(input, &writeOutputProps, &mergeOutputProps,
		func(ctx context.Context, args []string, st *build.State) error {
			logging.Infof(ctx, "have input %v", input)
			execute.Run(ctx, input)
			// actual build code here, build is already Start'd
			// input was parsed from build.Input.Properties
			writeOutputProps(&vpb.BumpVersionResponse{ErrorSummaryMarkdown: "winning!"})
			return nil // will mark the Build as SUCCESS
		},
	)
}
