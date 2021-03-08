// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package args

import (
	"time"

	buildapi "go.chromium.org/chromiumos/infra/proto/go/chromite/api"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"google.golang.org/protobuf/types/known/durationpb"
)

func basicInvocation() *steps.EnumerationResponse_AutotestInvocation {
	return &steps.EnumerationResponse_AutotestInvocation{
		Test: &buildapi.AutotestTest{
			ExecutionEnvironment: buildapi.AutotestTest_EXECUTION_ENVIRONMENT_CLIENT,
		},
	}
}

func setTestName(inv *steps.EnumerationResponse_AutotestInvocation, name string) {
	if inv.Test == nil {
		inv.Test = &buildapi.AutotestTest{}
	}
	inv.Test.Name = name
}

func setExecutionEnvironment(inv *steps.EnumerationResponse_AutotestInvocation, env buildapi.AutotestTest_ExecutionEnvironment) {
	if inv.Test == nil {
		inv.Test = &buildapi.AutotestTest{}
	}
	inv.Test.ExecutionEnvironment = env
}

func setTestKeyval(inv *steps.EnumerationResponse_AutotestInvocation, key string, value string) {
	if inv.ResultKeyvals == nil {
		inv.ResultKeyvals = make(map[string]string)
	}
	inv.ResultKeyvals[key] = value
}

func setTestArgs(inv *steps.EnumerationResponse_AutotestInvocation, testArgs string) {
	inv.TestArgs = testArgs
}

func setDisplayName(inv *steps.EnumerationResponse_AutotestInvocation, name string) {
	inv.DisplayName = name
}

func setBuild(p *test_platform.Request_Params, build string) {
	p.SoftwareDependencies = append(p.SoftwareDependencies,
		&test_platform.Request_Params_SoftwareDependency{
			Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{
				ChromeosBuild: build,
			},
		})
}

func setRequestKeyval(p *test_platform.Request_Params, key string, value string) {
	if p.Decorations == nil {
		p.Decorations = &test_platform.Request_Params_Decorations{}
	}
	if p.Decorations.AutotestKeyvals == nil {
		p.Decorations.AutotestKeyvals = make(map[string]string)
	}
	p.Decorations.AutotestKeyvals[key] = value
}

func setRequestMaximumDuration(p *test_platform.Request_Params, maximumDuration time.Duration) {
	if p.Time == nil {
		p.Time = &test_platform.Request_Params_Time{}
	}
	p.Time.MaximumDuration = durationpb.New(maximumDuration)
}
