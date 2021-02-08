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
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner/steps"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/luciexe/build"
)

const (
	dummyTestID = "original test"
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

func marshalInput(ctx context.Context, input *steps.RunTestsRequest) (tests map[string]*skylab_test_runner.Request_Test, err error) {
	step, ctx := build.StartStep(ctx, "marshal input")
	defer func() { step.End(err) }()
	tests = make(map[string]*skylab_test_runner.Request_Test)
	for k, v := range input.GetRequest().GetTests() {
		if k == dummyTestID {
			return tests, errors.Reason("cannot use reserved key %s as test key in input", dummyTestID).Err()
		}
		tests[k] = v
	}
	if input.GetRequest().GetTest() != nil {
		tests[dummyTestID] = input.GetRequest().GetTest()
	}
	return tests, nil
}

func parseEnvironment(ctx context.Context) (hostName string, err error) {
	step, ctx := build.StartStep(ctx, "parse environment information")
	defer func() { step.End(err) }()
	botID := os.Getenv("SWARMING_BOT_ID")
	prefix := "crossk-"
	if !strings.HasPrefix(botID, "crossk-") {
		return "", errors.Reason("expected SWARMING_BOT_ID to start with %v, instead %v", prefix, botID).Err()
	}
	hn := strings.TrimPrefix(botID, prefix)
	step.SetSummaryMarkdown("hostname: " + hn)
	return hn, nil
}

func executionSteps(ctx context.Context, tests map[string]*skylab_test_runner.Request_Test) (err error) {
	step, ctx := build.StartStep(ctx, "execution steps")
	defer func() { step.End(err) }()

	// TODO(https://crbug.com/1165962) publish to result flow here

	_, err = parseEnvironment(ctx)
	if err != nil {
		return errors.Annotate(err, "failed to parse environment information").Err()
	}

	return nil
}

// Run executes the core logic for cros_test_runner.
func Run(ctx context.Context, input *steps.RunTestsRequest) (err error) {
	if err = logInputs(ctx, input); err != nil {
		return err
	}

	tests, err := marshalInput(ctx, input)
	if err != nil {
		return errors.Annotate(err, "failed to marshal input").Err()
	}

	if err = executionSteps(ctx, tests); err != nil {
		err = errors.Annotate(err, "execution steps failed").Err()
	}

	return nil
}
