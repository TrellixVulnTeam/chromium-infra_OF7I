// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package args

import (
	"time"

	build_api "go.chromium.org/chromiumos/infra/proto/go/chromite/api"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
)

var noDeadline time.Time

func basicInvocation() *steps.EnumerationResponse_AutotestInvocation {
	return &steps.EnumerationResponse_AutotestInvocation{
		Test: &build_api.AutotestTest{
			ExecutionEnvironment: build_api.AutotestTest_EXECUTION_ENVIRONMENT_CLIENT,
		},
	}
}

func setTestName(inv *steps.EnumerationResponse_AutotestInvocation, name string) {
	if inv.Test == nil {
		inv.Test = &build_api.AutotestTest{}
	}
	inv.Test.Name = name
}

func setExecutionEnvironment(inv *steps.EnumerationResponse_AutotestInvocation, env build_api.AutotestTest_ExecutionEnvironment) {
	if inv.Test == nil {
		inv.Test = &build_api.AutotestTest{}
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

func setFWRO(p *test_platform.Request_Params, ver string) {
	p.SoftwareDependencies = append(p.SoftwareDependencies,
		&test_platform.Request_Params_SoftwareDependency{
			Dep: &test_platform.Request_Params_SoftwareDependency_RoFirmwareBuild{
				RoFirmwareBuild: ver,
			},
		})
}

func setFWRW(p *test_platform.Request_Params, ver string) {
	p.SoftwareDependencies = append(p.SoftwareDependencies,
		&test_platform.Request_Params_SoftwareDependency{
			Dep: &test_platform.Request_Params_SoftwareDependency_RwFirmwareBuild{
				RwFirmwareBuild: ver,
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

func setEnableSynchronousOffload(p *test_platform.Request_Params) {
	if p.Migrations == nil {
		p.Migrations = &test_platform.Request_Params_Migrations{}
	}
	p.Migrations.EnableSynchronousOffload = true
}

func unsetEnableSynchronousOffload(p *test_platform.Request_Params) {
	if p.Migrations == nil {
		p.Migrations = &test_platform.Request_Params_Migrations{}
	}
	p.Migrations.EnableSynchronousOffload = false
}

func unsetMigrationsConfig(p *test_platform.Request_Params) {
	p.Migrations = nil
}
