// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execution_test

// This file contains tests for behaviour when the user input is invalid in
// some way.

import (
	"context"
	"fmt"
	"testing"

	trservice "infra/cmd/cros_test_platform/internal/execution/testrunner/service"

	"github.com/golang/protobuf/ptypes/duration"
	. "github.com/smartystreets/goconvey/convey"
	buildapi "go.chromium.org/chromiumos/infra/proto/go/chromite/api"
	"go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
)

func TestLaunchForNonExistentBot(t *testing.T) {
	Convey("In an execution with one invocation but not bots", t, func() {
		trc := &trservice.CallCountingClientWrapper{
			Client: trservice.NewBotsAwareFakeClient(),
		}
		resps, err := runWithParams(
			context.Background(),
			trc,
			&test_platform.Request_Params{
				FreeformAttributes: &test_platform.Request_Params_FreeformAttributes{
					SwarmingDimensions: []string{"freeform-key:freeform-value"},
				},
				// Irrelevant required fields follow.
				Scheduling: &test_platform.Request_Params_Scheduling{
					Pool: &test_platform.Request_Params_Scheduling_ManagedPool_{
						ManagedPool: test_platform.Request_Params_Scheduling_MANAGED_POOL_CQ,
					},
					Priority: 79,
				},
				Time: &test_platform.Request_Params_Time{
					MaximumDuration: &duration.Duration{Seconds: 60},
				},
			},
			[]*steps.EnumerationResponse_AutotestInvocation{
				clientTestInvocation("", ""),
			},
		)
		So(err, ShouldBeNil)

		resp := extractSingleResponse(resps)

		Convey("then task result is complete with unspecified verdict.", func() {
			So(resp.TaskResults, ShouldHaveLength, 1)
			tr := resp.TaskResults[0]
			So(tr.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_REJECTED)
			So(tr.State.Verdict, ShouldEqual, test_platform.TaskState_VERDICT_UNSPECIFIED)
			So(tr.RejectedTaskDimensions, ShouldContainKey, "freeform-key")
			So(tr.RejectedDimensions[1].Key, ShouldEqual, "freeform-key")
		})
		Convey("and overall result is complete with failed verdict.", func() {
			So(resp.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_COMPLETED)
			So(resp.State.Verdict, ShouldEqual, test_platform.TaskState_VERDICT_FAILED)
		})
		Convey("and no skylab tasks are created.", func() {
			So(trc.CallCounts.LaunchTask, ShouldEqual, 0)
			So(trc.CallCounts.FetchResults, ShouldEqual, 0)
		})
	})
}

func TestRequestShouldNotSetBothQSAccountAndPriority(t *testing.T) {
	Convey("Given a client test with both quota account and priority set", t, func() {
		params := basicParams()
		params.Scheduling = &test_platform.Request_Params_Scheduling{
			Pool: &test_platform.Request_Params_Scheduling_UnmanagedPool{
				UnmanagedPool: "foo-pool",
			},
			QsAccount: "foo-account",
			Priority:  50,
		}
		Convey("The test should end up with a panic.", func() {
			So(
				func() {
					runWithParams(
						context.Background(),
						trservice.StubClient{},
						params,
						[]*steps.EnumerationResponse_AutotestInvocation{
							serverTestInvocation("name1", ""),
						},
					)
				},
				ShouldPanic,
			)
		})
	})
}

func TestIncompatibleDependencies(t *testing.T) {

	Convey("In testing context", t, func() {
		cases := []struct {
			Tag    string
			Params *test_platform.Request_Params
			Invs   []*steps.EnumerationResponse_AutotestInvocation
		}{
			{
				Tag: "incompatible build target between enumeration and request",
				Params: &test_platform.Request_Params{
					SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
						BuildTarget: &chromiumos.BuildTarget{Name: "requested"},
					},
					Time: &test_platform.Request_Params_Time{
						MaximumDuration: &duration.Duration{Seconds: 3600},
					},
				},
				Invs: []*steps.EnumerationResponse_AutotestInvocation{
					testInvocationWithDependency("some_test", "board:enumerated"),
				},
			},
			{
				Tag: "incompatible model between enumeration and request",
				Params: &test_platform.Request_Params{
					HardwareAttributes: &test_platform.Request_Params_HardwareAttributes{
						Model: "requested",
					},
					Time: &test_platform.Request_Params_Time{
						MaximumDuration: &duration.Duration{Seconds: 3600},
					},
				},
				Invs: []*steps.EnumerationResponse_AutotestInvocation{
					testInvocationWithDependency("some_test", "model:enumerated"),
				},
			},
			{
				Tag:    "unsupported dependencies",
				Params: basicParams(),
				Invs: []*steps.EnumerationResponse_AutotestInvocation{
					testInvocationWithDependency("some_test", "some_unsupported_dependency"),
				},
			},
		}

		for _, c := range cases {
			Convey(fmt.Sprintf("with %s", c.Tag), func() {
				trClient := &trservice.CallCountingClientWrapper{
					Client: trservice.StubClient{},
				}
				resps, err := runWithParams(
					context.Background(),
					trClient,
					c.Params,
					c.Invs,
				)
				So(err, ShouldBeNil)
				resp := extractSingleResponse(resps)

				Convey("then task result is rejected with unspecified verdict.", func() {
					So(resp.TaskResults, ShouldHaveLength, 1)
					tr := resp.TaskResults[0]
					So(tr.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_REJECTED)
					So(tr.State.Verdict, ShouldEqual, test_platform.TaskState_VERDICT_UNSPECIFIED)

				})
				Convey("and overall result is complete with failed verdict.", func() {
					So(resp.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_COMPLETED)
					So(resp.State.Verdict, ShouldEqual, test_platform.TaskState_VERDICT_FAILED)
				})
				Convey("and no skylab swarming tasks are created.", func() {
					So(trClient.CallCounts.LaunchTask, ShouldEqual, 0)
					So(trClient.CallCounts.FetchResults, ShouldEqual, 0)
				})
			})
		}
	})
}

func testInvocationWithDependency(name string, deps ...string) *steps.EnumerationResponse_AutotestInvocation {
	inv := steps.EnumerationResponse_AutotestInvocation{
		Test: &buildapi.AutotestTest{
			Name:                 name,
			ExecutionEnvironment: buildapi.AutotestTest_EXECUTION_ENVIRONMENT_SERVER,
		},
	}
	for _, d := range deps {
		inv.Test.Dependencies = append(inv.Test.Dependencies, &buildapi.AutotestTaskDependency{Label: d})
	}
	return &inv
}
