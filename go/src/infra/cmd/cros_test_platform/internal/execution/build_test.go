// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execution_test

// This file contains tests to cover the update of the Build proto during an
// execution.

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"infra/cmd/cros_test_platform/internal/execution"
	trservice "infra/cmd/cros_test_platform/internal/execution/testrunner/service"

	"github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/luciexe/exe"
)

func TestFinalBuildForSingleInvocation(t *testing.T) {
	Convey("For a run with one request with one invocation", t, func() {
		ba := newBuildAccumulator()
		_, err := runWithBuildAccumulator(
			context.Background(),
			stubTestRunnerClientWithCannedURL{
				Client:    trservice.NewStubClientWithSuccessfulTasks(),
				CannedURL: exampleTestRunnerURL,
			},
			ba,
			&steps.ExecuteRequests{
				TaggedRequests: map[string]*steps.ExecuteRequest{
					"request-with-single-invocation": {
						RequestParams: basicParams(),
						Enumeration: &steps.EnumerationResponse{
							AutotestInvocations: []*steps.EnumerationResponse_AutotestInvocation{
								clientTestInvocation("first-invocation", ""),
							},
						},
					},
				},
			},
		)
		So(err, ShouldBeNil)

		b := ba.GetLatestBuild()
		So(b, ShouldNotBeNil)
		So(b.GetSteps(), ShouldHaveLength, 2)

		rs := stepForRequest(b, "request-with-single-invocation")
		So(rs, ShouldNotBeNil)

		is := stepForInvocation(b, "first-invocation")
		So(is, ShouldNotBeNil)
		So(is.Name, ShouldContainSubstring, "request-with-single-invocation")
		markdownContainsURL(is.GetSummaryMarkdown(), "latest attempt", exampleTestRunnerURL)
	})
}

func TestFinalBuildForTwoInvocations(t *testing.T) {
	Convey("For a run with one request with two invocations", t, func() {
		ba := newBuildAccumulator()
		_, err := runWithBuildAccumulator(
			context.Background(),
			stubTestRunnerClientWithCannedURL{
				Client:    trservice.NewStubClientWithSuccessfulTasks(),
				CannedURL: exampleTestRunnerURL,
			},
			ba,
			&steps.ExecuteRequests{
				TaggedRequests: map[string]*steps.ExecuteRequest{
					"request-with-two-invocations": {
						RequestParams: basicParams(),
						Enumeration: &steps.EnumerationResponse{
							AutotestInvocations: []*steps.EnumerationResponse_AutotestInvocation{
								clientTestInvocation("first-invocation", ""),
								clientTestInvocation("second-invocation", ""),
							},
						},
					},
				},
			},
		)
		So(err, ShouldBeNil)

		b := ba.GetLatestBuild()
		So(b, ShouldNotBeNil)
		So(b.GetSteps(), ShouldHaveLength, 3)

		rs := stepForRequest(b, "request-with-two-invocations")
		So(rs, ShouldNotBeNil)

		is := stepForInvocation(b, "first-invocation")
		So(is, ShouldNotBeNil)
		So(is.Name, ShouldContainSubstring, "request-with-two-invocations")
		markdownContainsURL(is.GetSummaryMarkdown(), "latest attempt", exampleTestRunnerURL)

		is = stepForInvocation(b, "second-invocation")
		So(is, ShouldNotBeNil)
		So(is.Name, ShouldContainSubstring, "request-with-two-invocations")
		markdownContainsURL(is.GetSummaryMarkdown(), "latest attempt", exampleTestRunnerURL)
	})
}

func TestFinalBuildForTwoRequests(t *testing.T) {
	Convey("For a run with two requests with one invocation each", t, func() {
		ba := newBuildAccumulator()
		_, err := runWithBuildAccumulator(
			context.Background(),
			stubTestRunnerClientWithCannedURL{
				Client:    trservice.NewStubClientWithSuccessfulTasks(),
				CannedURL: exampleTestRunnerURL,
			},
			ba,
			&steps.ExecuteRequests{
				TaggedRequests: map[string]*steps.ExecuteRequest{
					"first-request": {
						RequestParams: basicParams(),
						Enumeration: &steps.EnumerationResponse{
							AutotestInvocations: []*steps.EnumerationResponse_AutotestInvocation{
								clientTestInvocation("first-request-invocation", ""),
							},
						},
					},
					"second-request": {
						RequestParams: basicParams(),
						Enumeration: &steps.EnumerationResponse{
							AutotestInvocations: []*steps.EnumerationResponse_AutotestInvocation{
								clientTestInvocation("second-request-invocation", ""),
							},
						},
					},
				},
			},
		)
		So(err, ShouldBeNil)

		b := ba.GetLatestBuild()
		So(b, ShouldNotBeNil)
		So(b.GetSteps(), ShouldHaveLength, 4)

		rs := stepForRequest(b, "first-request")
		So(rs, ShouldNotBeNil)
		is := stepForInvocation(b, "first-request-invocation")
		So(is, ShouldNotBeNil)
		So(is.Name, ShouldContainSubstring, "first-request")
		markdownContainsURL(is.GetSummaryMarkdown(), "latest attempt", exampleTestRunnerURL)

		rs = stepForRequest(b, "second-request")
		So(rs, ShouldNotBeNil)
		is = stepForInvocation(b, "second-request-invocation")
		So(is, ShouldNotBeNil)
		So(is.Name, ShouldContainSubstring, "second-request")
		markdownContainsURL(is.GetSummaryMarkdown(), "latest attempt", exampleTestRunnerURL)
	})
}

func TestFinalBuildForSingleInvocationWithRetries(t *testing.T) {
	Convey("For a run with one request with one invocation that needs 1 retry", t, func() {
		params := basicParams()
		params.Retry = &test_platform.Request_Params_Retry{
			Allow: true,
			Max:   1,
		}
		inv := clientTestInvocation("failing-invocation", "")
		inv.Test.AllowRetries = true
		req := &steps.ExecuteRequests{
			TaggedRequests: map[string]*steps.ExecuteRequest{
				"request-with-one-retry-allowed": {
					RequestParams: params,
					Enumeration: &steps.EnumerationResponse{
						AutotestInvocations: []*steps.EnumerationResponse_AutotestInvocation{inv},
					},
				},
			},
		}

		ba := newBuildAccumulator()
		_, err := runWithBuildAccumulator(
			setFakeTimeWithImmediateTimeout(context.Background()),
			stubTestRunnerClientWithCannedURL{
				Client:    trservice.NewStubClientWithFailedTasks(),
				CannedURL: exampleTestRunnerURL,
			},
			ba,
			req,
		)
		So(err, ShouldBeNil)

		b := ba.GetLatestBuild()
		So(b, ShouldNotBeNil)
		So(b.GetSteps(), ShouldHaveLength, 2)

		is := stepForInvocation(b, "failing-invocation")
		So(is, ShouldNotBeNil)

		markdownContainsURL(is.GetSummaryMarkdown(), "latest attempt", exampleTestRunnerURL)
		markdownContainsURL(is.GetSummaryMarkdown(), "1", exampleTestRunnerURL)
	})
}

func TestBuildUpdatesWithRetries(t *testing.T) {
	Convey("Compared to a run without retries", t, func() {
		inv := clientTestInvocation("failing-invocation", "")
		inv.Test.AllowRetries = true
		e := &steps.EnumerationResponse{
			AutotestInvocations: []*steps.EnumerationResponse_AutotestInvocation{inv},
		}
		req := &steps.ExecuteRequests{
			TaggedRequests: map[string]*steps.ExecuteRequest{
				"no-retry": {
					RequestParams: basicParams(),
					Enumeration:   e,
				},
			},
		}

		ba := newBuildAccumulator()
		_, err := runWithBuildAccumulator(
			setFakeTimeWithImmediateTimeout(context.Background()),
			stubTestRunnerClientWithCannedURL{
				Client:    trservice.NewStubClientWithFailedTasks(),
				CannedURL: exampleTestRunnerURL,
			},
			ba,
			req,
		)
		So(err, ShouldBeNil)
		noRetryUpdateCount := len(ba.Sent)
		_ = noRetryUpdateCount

		Convey("a run with a retry should send more updates", func() {
			params := basicParams()
			params.Retry = &test_platform.Request_Params_Retry{
				Allow: true,
				Max:   1,
			}
			req := &steps.ExecuteRequests{
				TaggedRequests: map[string]*steps.ExecuteRequest{
					"one-retry": {
						RequestParams: params,
						Enumeration:   e,
					},
				},
			}

			ba := newBuildAccumulator()
			_, err := runWithBuildAccumulator(
				setFakeTimeWithImmediateTimeout(context.Background()),
				stubTestRunnerClientWithCannedURL{
					Client:    trservice.NewStubClientWithFailedTasks(),
					CannedURL: exampleTestRunnerURL,
				},
				ba,
				req,
			)
			So(err, ShouldBeNil)
			oneRetryUpdateCount := len(ba.Sent)

			So(oneRetryUpdateCount, ShouldBeGreaterThan, noRetryUpdateCount)
		})
	})
}

const exampleTestRunnerURL = "https://ci.chromium.org/p/chromeos/builders/test_runner/test_runner/b8872341436802087200"

// buildAccumulator supports a Send method to accumulate the bbpb.Build sent
// to the host application.
//
// Typical usage:
//
//   ba := newBuildAccumulator()
//   err := runWithBuildAccumulator(..., ba, ...)
//   So(err, ShouldBeNil)
//   So(ba.GetLatestBuild(), ShouldNotBeNil)
//   ...
type buildAccumulator struct {
	Input *bbpb.Build
	Sent  []*bbpb.Build
}

func newBuildAccumulator() *buildAccumulator {
	return &buildAccumulator{
		Input: &bbpb.Build{},
		Sent:  []*bbpb.Build{},
	}
}

func (s *buildAccumulator) Send() {
	s.Sent = append(s.Sent, proto.Clone(s.Input).(*bbpb.Build))
}

func (s *buildAccumulator) GetLatestBuild() *bbpb.Build {
	if len(s.Sent) == 0 {
		return nil
	}
	return s.Sent[len(s.Sent)-1]
}

func runWithBuildAccumulator(ctx context.Context, skylab trservice.Client, ba *buildAccumulator, request *steps.ExecuteRequests) (map[string]*steps.ExecuteResponse, error) {
	args := execution.Args{
		Build:   ba.Input,
		Send:    exe.BuildSender(ba.Send),
		Request: request,
		WorkerConfig: &config.Config_SkylabWorker{
			LuciProject: "foo-luci-project",
			LogDogHost:  "foo-logdog-host",
		},
		ParentTaskID: "foo-parent-task-id",
		Deadline:     time.Now().Add(time.Hour),
	}
	return execution.Run(ctx, skylab, args)
}

type stubTestRunnerClientWithCannedURL struct {
	trservice.Client
	CannedURL string
}

// URL implements Client interface.
func (c stubTestRunnerClientWithCannedURL) URL(trservice.TaskReference) string {
	return c.CannedURL
}

// stepForRequest returns the first step for a request with the given name.
//
// Returns nil if no such step is found.
func stepForRequest(build *bbpb.Build, name string) *bbpb.Step {
	for _, s := range build.Steps {
		if isRequestStep(s.GetName()) && strings.Contains(s.GetName(), name) {
			return s
		}
	}
	return nil
}

// stepForInvocation returns the first step for an invocation with the given
// name.
//
// Returns nil if no such step is found.
func stepForInvocation(build *bbpb.Build, name string) *bbpb.Step {
	for _, s := range build.Steps {
		if isInvocationStep(s.GetName()) && strings.Contains(s.GetName(), name) {
			return s
		}
	}
	return nil
}

var (
	requestStepRe    = regexp.MustCompile(`\s*request.*\|\s*invocation.*`)
	invocationStepRe = regexp.MustCompile(`.*|\s*invocation.*`)
)

func isRequestStep(name string) bool {
	return requestStepRe.Match([]byte(name))
}

func isInvocationStep(name string) bool {
	return invocationStepRe.Match([]byte(name))
}

func markdownContainsURL(md string, target string, url string) {
	So(md, ShouldContainSubstring, fmt.Sprintf("[%s](%s)", target, url))
}
