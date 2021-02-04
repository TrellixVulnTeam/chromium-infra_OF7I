// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execute

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner/steps"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/luciexe/build"
)

func logInputs(ctx context.Context, input *steps.RunTestsRequest) (err error) {
	step, ctx := build.StartStep(ctx, "inputs")
	defer func() { step.End(err) }()
	step.SetSummaryMarkdown(fmt.Sprintf("* [parent CTP](https://cr-buildbucket.appspot.com/build/%d)", input.Request.GetParentBuildId()))
	req := step.Log("input proto")
	marsh := jsonpb.Marshaler{Indent: "  "}
	if err = marsh.Marshal(req, input); err != nil {
		return errors.Annotate(err, "failed to marshal proto").Err()
	}
	return nil
}

// Run executes the core logic for cros_test_runner.
func Run(ctx context.Context, input *steps.RunTestsRequest) (err error) {
	if err = logInputs(ctx, input); err != nil {
		return err
	}

	botID := os.Getenv("SWARMING_BOT_ID")
	prefix := "crossk-"
	if !strings.HasPrefix(botID, "crossk-") {
		return fmt.Errorf("expected SWARMING_BOT_ID to start with %v, instead %v", prefix, botID)
	}
	_ = strings.TrimPrefix(botID, prefix)
	return nil
}
