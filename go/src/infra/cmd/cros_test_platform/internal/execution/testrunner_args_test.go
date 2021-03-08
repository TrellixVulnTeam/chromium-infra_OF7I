// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execution_test

// This file contains tests to ensure that testrunner builds are created with
// the right arguments.

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	trservice "infra/cmd/cros_test_platform/internal/execution/testrunner/service"
	"infra/libs/skylab/inventory"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/chromite/api"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
)

func TestRequestArguments(t *testing.T) {
	Convey("Given a server test with autotest labels", t, func() {
		trClient := &trservice.ArgsCollectingClientWrapper{
			Client: trservice.StubClient{},
		}

		inv := serverTestInvocation("name1", "foo-arg1 foo-arg2")
		addAutotestDependency(inv, "cr50:pvt")
		addAutotestDependency(inv, "cleanup-reboot")
		inv.DisplayName = "given_name"
		invs := []*steps.EnumerationResponse_AutotestInvocation{inv}

		_, err := runWithDefaults(
			context.Background(),
			trClient,
			invs,
		)
		So(err, ShouldBeNil)

		Convey("the launched task request should have correct parameters.", func() {
			So(trClient.Calls.LaunchTask, ShouldHaveLength, 1)
			launchArgs := trClient.Calls.LaunchTask[0].Args

			So(launchArgs.SwarmingTags, ShouldContain, "parent_buildbucket_id:42")
			So(launchArgs.SwarmingTags, ShouldContain, "luci_project:foo-luci-project")
			So(launchArgs.SwarmingTags, ShouldContain, "foo-tag1")
			So(launchArgs.SwarmingTags, ShouldContain, "foo-tag2")
			So(launchArgs.ParentTaskID, ShouldEqual, "foo-parent-task-id")
			So(launchArgs.ParentRequestUID, ShouldEqual, "TestPlanRuns/42/12345678/foo")

			So(launchArgs.Priority, ShouldEqual, 79)

			prefix := "log_location:"
			var logdogURL string
			matchingTags := 0
			for _, tag := range launchArgs.SwarmingTags {
				if strings.HasPrefix(tag, prefix) {
					matchingTags++
					So(tag, ShouldEndWith, "+/annotations")

					logdogURL = strings.TrimPrefix(tag, "log_location:")
				}
			}
			So(matchingTags, ShouldEqual, 1)
			So(logdogURL, ShouldStartWith, "logdog://foo-logdog-host/foo-luci-project/skylab/")
			So(logdogURL, ShouldEndWith, "/+/annotations")

			So(launchArgs.Cmd.TaskName, ShouldEqual, "name1")
			So(launchArgs.Cmd.ClientTest, ShouldBeFalse)

			// Logdog annotation url argument should match the associated tag's url.
			So(launchArgs.Cmd.LogDogAnnotationURL, ShouldEqual, logdogURL)

			So(launchArgs.Cmd.TestArgs, ShouldEqual, "foo-arg1 foo-arg2")

			So(launchArgs.Cmd.Keyvals["k1"], ShouldEqual, "v1")
			So(launchArgs.Cmd.Keyvals["parent_job_id"], ShouldEqual, "foo-parent-task-id")
			So(launchArgs.Cmd.Keyvals["label"], ShouldEqual, "given_name")

			So(launchArgs.ProvisionableDimensions, ShouldHaveLength, 3)
			So(launchArgs.ProvisionableDimensions, ShouldContain, "provisionable-cros-version:foo-build")
			So(launchArgs.ProvisionableDimensions, ShouldContain, "provisionable-fwro-version:foo-ro-firmware")
			So(launchArgs.ProvisionableDimensions, ShouldContain, "provisionable-fwrw-version:foo-rw-firmware")

			So(launchArgs.ProvisionableDimensionExpiration, ShouldEqual, time.Minute)

			So(launchArgs.SchedulableLabels.GetCr50Phase(), ShouldEqual, inventory.SchedulableLabels_CR50_PHASE_PVT)
			So(launchArgs.SchedulableLabels.GetModel(), ShouldEqual, "foo-model")
			So(launchArgs.SchedulableLabels.GetBoard(), ShouldEqual, "foo-board")
			So(launchArgs.SchedulableLabels.GetCriticalPools(), ShouldHaveLength, 1)
			So(launchArgs.SchedulableLabels.GetCriticalPools()[0], ShouldEqual, inventory.SchedulableLabels_DUT_POOL_CQ)

			So(launchArgs.Dimensions, ShouldHaveLength, 1)
			So(launchArgs.Dimensions, ShouldContain, "freeform-key:freeform-value")
		})
	})
}

func TestInvocationKeyvals(t *testing.T) {
	Convey("Given an enumeration with a suite keyval", t, func() {
		trClient := &trservice.ArgsCollectingClientWrapper{
			Client: trservice.StubClient{},
		}
		invs := []*steps.EnumerationResponse_AutotestInvocation{
			{
				Test: &api.AutotestTest{
					Name:                 "someTest",
					ExecutionEnvironment: api.AutotestTest_EXECUTION_ENVIRONMENT_CLIENT,
				},
				ResultKeyvals: map[string]string{
					"suite": "someSuite",
				},
			},
		}

		Convey("and a request without keyvals", func() {
			p := basicParams()
			p.Decorations = nil
			_, err := runWithDefaults(context.Background(), trClient, invs)
			So(err, ShouldBeNil)

			Convey("created command includes invocation suite keyval", func() {
				So(trClient.Calls.LaunchTask, ShouldHaveLength, 1)
				launchArgs := trClient.Calls.LaunchTask[0].Args
				flatCommand := strings.Join(launchArgs.Cmd.Args(), " ")
				keyvals := extractKeyvalsArgument(flatCommand)
				So(keyvals, ShouldContainSubstring, `"suite":"someSuite"`)
				So(keyvals, ShouldContainSubstring, `"label":"foo-build/someSuite/someTest"`)
			})
		})

		Convey("and a request with different suite keyvals", func() {
			p := basicParams()
			p.Decorations = &test_platform.Request_Params_Decorations{
				AutotestKeyvals: map[string]string{
					"suite": "someOtherSuite",
				},
			}

			_, err := runWithParams(context.Background(), trClient, p, invs)
			So(err, ShouldBeNil)

			Convey("created command includes request suite keyval", func() {
				So(trClient.Calls.LaunchTask, ShouldHaveLength, 1)
				launchArgs := trClient.Calls.LaunchTask[0].Args
				flatCommand := strings.Join(launchArgs.Cmd.Args(), " ")
				keyvals := extractKeyvalsArgument(flatCommand)
				So(keyvals, ShouldContainSubstring, `"suite":"someOtherSuite"`)
				So(keyvals, ShouldContainSubstring, `"label":"foo-build/someOtherSuite/someTest"`)
			})
		})
	})
}

func TestKeyvalsAcrossTestRuns(t *testing.T) {
	Convey("Given a request with a suite keyval", t, func() {
		p := basicParams()
		p.Decorations = &test_platform.Request_Params_Decorations{
			AutotestKeyvals: map[string]string{
				"suite": "someSuite",
			},
		}

		Convey("and two enumerations with different test names", func() {

			invs := []*steps.EnumerationResponse_AutotestInvocation{
				{
					Test: &api.AutotestTest{
						Name:                 "firstTest",
						ExecutionEnvironment: api.AutotestTest_EXECUTION_ENVIRONMENT_CLIENT,
					},
				},
				{
					Test: &api.AutotestTest{
						Name:                 "secondTest",
						ExecutionEnvironment: api.AutotestTest_EXECUTION_ENVIRONMENT_CLIENT,
					},
				},
			}

			Convey("created commands include common suite keyval and different label keyvals", func() {
				trClient := &trservice.ArgsCollectingClientWrapper{
					Client: trservice.StubClient{},
				}
				_, err := runWithParams(context.Background(), trClient, p, invs)
				So(err, ShouldBeNil)

				So(trClient.Calls.LaunchTask, ShouldHaveLength, 2)
				cmd := make([]string, 2)
				for i, lt := range trClient.Calls.LaunchTask {
					cmd[i] = strings.Join(lt.Args.Cmd.Args(), " ")
				}
				kv0 := extractKeyvalsArgument(cmd[0])
				So(kv0, ShouldContainSubstring, `"suite":"someSuite"`)
				So(kv0, ShouldContainSubstring, `"label":"foo-build/someSuite/firstTest"`)
				kv1 := extractKeyvalsArgument(cmd[1])
				So(kv1, ShouldContainSubstring, `"suite":"someSuite"`)
				So(kv1, ShouldContainSubstring, `"label":"foo-build/someSuite/secondTest"`)
			})
		})
	})
}

func TestClientTestArg(t *testing.T) {
	Convey("Given a client test", t, func() {
		trClient := &trservice.ArgsCollectingClientWrapper{
			Client: trservice.StubClient{},
		}
		_, err := runWithDefaults(
			context.Background(),
			trClient,
			[]*steps.EnumerationResponse_AutotestInvocation{
				clientTestInvocation("name1", ""),
			},
		)
		So(err, ShouldBeNil)

		Convey("the launched task request should have correct parameters.", func() {
			So(trClient.Calls.LaunchTask, ShouldHaveLength, 1)
			So(trClient.Calls.LaunchTask[0].Args.Cmd.ClientTest, ShouldBeTrue)
		})
	})
}

func TestQuotaSchedulerAccountOnQSAccount(t *testing.T) {
	Convey("Given a client test and a quota account", t, func() {
		trClient := &trservice.ArgsCollectingClientWrapper{
			Client: trservice.StubClient{},
		}
		params := basicParams()
		params.Scheduling = &test_platform.Request_Params_Scheduling{
			Pool: &test_platform.Request_Params_Scheduling_UnmanagedPool{
				UnmanagedPool: "foo-pool",
			},
			QsAccount: "foo-account",
		}
		_, err := runWithParams(
			context.Background(),
			trClient,
			params,
			[]*steps.EnumerationResponse_AutotestInvocation{
				serverTestInvocation("name1", ""),
			},
		)
		So(err, ShouldBeNil)

		Convey("the launched task request should have a tag specifying the correct quota account and run in foo-pool.", func() {
			So(trClient.Calls.LaunchTask, ShouldHaveLength, 1)
			launchArgs := trClient.Calls.LaunchTask[0].Args
			So(launchArgs.SwarmingTags, ShouldContain, "qs_account:foo-account")
			So(launchArgs.SchedulableLabels.GetSelfServePools(), ShouldHaveLength, 1)
			So(launchArgs.SchedulableLabels.GetCriticalPools(), ShouldHaveLength, 0)
			So(launchArgs.SchedulableLabels.GetSelfServePools()[0], ShouldEqual, "foo-pool")
		})
	})
}

func TestReservedTagShouldNotBeSetByUsers(t *testing.T) {
	Convey("Given a client test and a fake quota account set by user", t, func() {
		trClient := &trservice.ArgsCollectingClientWrapper{
			Client: trservice.StubClient{},
		}
		params := basicParams()
		params.Scheduling = &test_platform.Request_Params_Scheduling{
			Pool: &test_platform.Request_Params_Scheduling_ManagedPool_{
				ManagedPool: test_platform.Request_Params_Scheduling_MANAGED_POOL_QUOTA,
			},
			QsAccount: "real-account",
		}
		params.Decorations = &test_platform.Request_Params_Decorations{
			Tags: []string{"qs_account:fake-account"},
		}

		_, err := runWithParams(
			context.Background(),
			trClient,
			params,
			[]*steps.EnumerationResponse_AutotestInvocation{
				serverTestInvocation("name1", ""),
			},
		)
		So(err, ShouldBeNil)

		Convey("the launched task request should have a tag specifying the correct quota account and run in the quota pool.", func() {
			So(trClient.Calls.LaunchTask, ShouldHaveLength, 1)
			launchArgs := trClient.Calls.LaunchTask[0].Args
			So(launchArgs.SwarmingTags, ShouldContain, "qs_account:real-account")
			So(launchArgs.SchedulableLabels.GetSelfServePools(), ShouldHaveLength, 0)
			So(launchArgs.SchedulableLabels.GetCriticalPools(), ShouldHaveLength, 1)
			So(launchArgs.SchedulableLabels.GetCriticalPools()[0], ShouldEqual, inventory.SchedulableLabels_DUT_POOL_QUOTA)
		})
	})
}

func TestUnmanagedPool(t *testing.T) {
	Convey("Given a client test and an unmanaged pool.", t, func() {
		trClient := &trservice.ArgsCollectingClientWrapper{
			Client: trservice.StubClient{},
		}
		params := basicParams()
		params.Scheduling.Pool = &test_platform.Request_Params_Scheduling_UnmanagedPool{
			UnmanagedPool: "foo-pool",
		}

		_, err := runWithParams(
			context.Background(),
			trClient,
			params,
			[]*steps.EnumerationResponse_AutotestInvocation{
				serverTestInvocation("name1", ""),
			},
		)
		So(err, ShouldBeNil)

		Convey("the launched task request run in the unmanaged pool.", func() {
			So(trClient.Calls.LaunchTask, ShouldHaveLength, 1)
			launchArgs := trClient.Calls.LaunchTask[0].Args
			So(launchArgs.SchedulableLabels.GetCriticalPools(), ShouldHaveLength, 0)
			So(launchArgs.SchedulableLabels.GetSelfServePools(), ShouldHaveLength, 1)
			So(launchArgs.SchedulableLabels.GetSelfServePools()[0], ShouldEqual, "foo-pool")
		})
	})
}

func addAutotestDependency(inv *steps.EnumerationResponse_AutotestInvocation, dep string) {
	inv.Test.Dependencies = append(inv.Test.Dependencies, &api.AutotestTaskDependency{Label: dep})
}

var keyvalsPattern = regexp.MustCompile(`\-keyvals\s*\{([\w\s":,-/]+)\}`)

func extractKeyvalsArgument(cmd string) string {
	ms := keyvalsPattern.FindAllStringSubmatch(cmd, -1)
	So(ms, ShouldHaveLength, 1)
	m := ms[0]
	// Guaranteed by the constant regexp definition.
	if len(m) != 2 {
		panic(fmt.Sprintf("Match %s of regexp %s has length %d, want 2", m, keyvalsPattern, len(m)))
	}
	return m[1]
}
