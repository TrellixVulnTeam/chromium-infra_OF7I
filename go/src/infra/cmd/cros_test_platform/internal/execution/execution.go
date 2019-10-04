// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package execution provides implementations for test execution runners.
package execution

import (
	"context"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/errors"

	"infra/cmd/cros_test_platform/internal/execution/internal/autotest"
	"infra/cmd/cros_test_platform/internal/execution/internal/skylab"
	"infra/cmd/cros_test_platform/internal/execution/isolate"
	"infra/cmd/cros_test_platform/internal/execution/swarming"
)

// Runner defines the interface implemented by Skylab or Autotest execution
// runners.
type Runner interface {
	LaunchAndWait(context.Context, swarming.Client, isolate.GetterFactory) error
	Responses(swarming.URLer) []*steps.ExecuteResponse
}

// NewSkylabRunner returns a Runner that will execute the given tests in
// the skylab environment.
func NewSkylabRunner(ctx context.Context, workerConfig *config.Config_SkylabWorker, parentTaskID string, requests []*steps.ExecuteRequest) (Runner, error) {
	ts := make([]*skylab.TaskSet, len(requests))
	for i, r := range requests {
		var err error
		ts[i], err = skylab.NewTaskSet(ctx, r.Enumeration.AutotestInvocations, r.RequestParams, workerConfig, parentTaskID)
		if err != nil {
			return nil, errors.Annotate(err, "new skylab runner").Err()
		}
	}
	return skylab.NewRunner(ts...), nil
}

// NewAutotestRunner returns a Runner that will execute the given tests in
// the autotest environment.
func NewAutotestRunner(tests []*steps.EnumerationResponse_AutotestInvocation, params *test_platform.Request_Params, config *config.Config_AutotestBackend) Runner {
	return autotest.New(tests, params, config)
}
