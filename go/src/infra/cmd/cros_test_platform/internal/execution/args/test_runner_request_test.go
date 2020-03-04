// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package args contains the logic for assembling all data required for
// creating an individual task request.
package args

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"

	. "github.com/smartystreets/goconvey/convey"

	build_api "go.chromium.org/chromiumos/infra/proto/go/chromite/api"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
)

var noDeadline time.Time

func TestProvisionableLabels(t *testing.T) {
	Convey("Given a test that specifies software dependencies", t, func() {
		ctx := context.Background()
		var params test_platform.Request_Params
		setBuild(&params, "foo-build")
		setFWRO(&params, "foo-ro-firmware")
		setFWRW(&params, "foo-rw-firmware")
		Convey("when generating a test runner request", func() {
			g := NewGenerator(basicInvocation(), &params, nil, "", noDeadline)
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("the provisionable labels match the software dependencies", func() {
				So(got.Prejob, ShouldNotBeNil)
				So(got.Prejob.ProvisionableLabels, ShouldNotBeNil)
				So(got.Prejob.ProvisionableLabels["cros-version"], ShouldEqual, "foo-build")
				So(got.Prejob.ProvisionableLabels["fwro-version"], ShouldEqual, "foo-ro-firmware")
				So(got.Prejob.ProvisionableLabels["fwrw-version"], ShouldEqual, "foo-rw-firmware")
			})
		})
	})
}

func TestClientTest(t *testing.T) {
	Convey("Given a client test", t, func() {
		ctx := context.Background()
		var inv steps.EnumerationResponse_AutotestInvocation
		setExecutionEnvironment(&inv, build_api.AutotestTest_EXECUTION_ENVIRONMENT_CLIENT)
		Convey("when generating a test runner request", func() {
			g := NewGenerator(&inv, &test_platform.Request_Params{}, nil, "", noDeadline)
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("it should be marked as such.", func() {
				So(got.Test, ShouldNotBeNil)
				So(got.Test.GetAutotest(), ShouldNotBeNil)
				So(got.Test.GetAutotest().IsClientTest, ShouldEqual, true)
			})
		})
	})
}

func TestServerTest(t *testing.T) {
	Convey("Given a server test", t, func() {
		ctx := context.Background()
		var inv steps.EnumerationResponse_AutotestInvocation
		setExecutionEnvironment(&inv, build_api.AutotestTest_EXECUTION_ENVIRONMENT_SERVER)
		Convey("when generating a test runner request", func() {
			g := NewGenerator(&inv, &test_platform.Request_Params{}, nil, "", noDeadline)
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("it should be marked as such.", func() {
				So(got.Test, ShouldNotBeNil)
				So(got.Test.GetAutotest(), ShouldNotBeNil)
				So(got.Test.GetAutotest().IsClientTest, ShouldEqual, false)
			})
		})
	})
}

func TestUnspecifiedTestEnvironment(t *testing.T) {
	Convey("Given a test that does not specify an environment", t, func() {
		ctx := context.Background()
		var inv steps.EnumerationResponse_AutotestInvocation
		setTestName(&inv, "foo-test")
		Convey("the test runner request generation fails.", func() {
			g := NewGenerator(&inv, &test_platform.Request_Params{}, nil, "", noDeadline)
			_, err := g.testRunnerRequest(ctx)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestTestName(t *testing.T) {
	Convey("Given a test", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		setTestName(inv, "foo-test")
		Convey("when generating a test runner request", func() {
			g := NewGenerator(inv, &test_platform.Request_Params{}, nil, "", noDeadline)
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("the test name is populated correctly.", func() {
				So(got.Test, ShouldNotBeNil)
				So(got.Test.GetAutotest(), ShouldNotBeNil)
				So(got.Test.GetAutotest().Name, ShouldEqual, "foo-test")
			})
		})
	})
}

func TestTestArgs(t *testing.T) {
	Convey("Given a request that specifies test args", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		setTestArgs(inv, "foo=bar baz=qux")
		Convey("when generating a test runner request", func() {
			g := NewGenerator(inv, &test_platform.Request_Params{}, nil, "", noDeadline)
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("the test args are propagated correctly.", func() {
				So(got.Test, ShouldNotBeNil)
				So(got.Test.GetAutotest(), ShouldNotBeNil)
				So(got.Test.GetAutotest().TestArgs, ShouldEqual, "foo=bar baz=qux")
			})
		})
	})
}

func TestTestLevelKeyval(t *testing.T) {
	Convey("Given a keyval inside the test invocation", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		setTestKeyval(inv, "key", "test-value")
		var params test_platform.Request_Params
		Convey("when generating a test runner request", func() {
			g := NewGenerator(inv, &params, nil, "", noDeadline)
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("the keyval is propagated.", func() {
				So(got.Test, ShouldNotBeNil)
				So(got.Test.GetAutotest(), ShouldNotBeNil)
				So(got.Test.GetAutotest().Keyvals, ShouldNotBeNil)
				So(got.Test.GetAutotest().Keyvals["key"], ShouldEqual, "test-value")
			})
		})
	})
}

func TestRequestLevelKeyval(t *testing.T) {
	Convey("Given request-wide keyval", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		setTestKeyval(inv, "key", "test-value")
		var params test_platform.Request_Params
		Convey("when generating a test runner request", func() {
			g := NewGenerator(inv, &params, nil, "", noDeadline)
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("the keyval is propagated.", func() {
				So(got.Test, ShouldNotBeNil)
				So(got.Test.GetAutotest(), ShouldNotBeNil)
				So(got.Test.GetAutotest().Keyvals, ShouldNotBeNil)
				So(got.Test.GetAutotest().Keyvals["key"], ShouldEqual, "test-value")
			})
		})
	})
}

func TestKeyvalOverride(t *testing.T) {
	Convey("Given keyvals with the same key in the invocation and request", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		setTestKeyval(inv, "ambiguous-key", "test-value")
		var params test_platform.Request_Params
		setRequestKeyval(&params, "ambiguous-key", "request-value")
		Convey("when generating a test runner request", func() {
			g := NewGenerator(inv, &params, nil, "", noDeadline)
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("the keyval from the request takes precedence.", func() {
				So(got.Test, ShouldNotBeNil)
				So(got.Test.GetAutotest(), ShouldNotBeNil)
				So(got.Test.GetAutotest().Keyvals, ShouldNotBeNil)
				So(got.Test.GetAutotest().Keyvals["ambiguous-key"], ShouldEqual, "request-value")
			})
		})
	})
}

func TestConstructedDisplayName(t *testing.T) {
	Convey("Given a request does not specify a display name", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		setTestName(inv, "foo-name")
		var params test_platform.Request_Params
		setBuild(&params, "foo-build")
		setRequestKeyval(&params, "suite", "foo-suite")
		Convey("when generating a test runner request", func() {
			g := NewGenerator(inv, &params, nil, "", noDeadline)
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("the display name is generated correctly.", func() {
				So(got.Test, ShouldNotBeNil)
				So(got.Test.GetAutotest(), ShouldNotBeNil)
				So(got.Test.GetAutotest().DisplayName, ShouldEqual, "foo-build/foo-suite/foo-name")
				So(got.Test.GetAutotest().Keyvals, ShouldNotBeNil)
				So(got.Test.GetAutotest().Keyvals["label"], ShouldEqual, "foo-build/foo-suite/foo-name")
			})
		})
	})
}

func TestExplicitDisplayName(t *testing.T) {
	Convey("Given a request that specifies a display name", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		setTestName(inv, "basic-name")
		setDisplayName(inv, "fancy-name")
		var params test_platform.Request_Params
		Convey("when generating a test runner request", func() {
			g := NewGenerator(inv, &params, nil, "", noDeadline)
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("the display name is propagated correctly.", func() {
				So(got.Test, ShouldNotBeNil)
				So(got.Test.GetAutotest(), ShouldNotBeNil)
				So(got.Test.GetAutotest().DisplayName, ShouldEqual, "fancy-name")
				So(got.Test.GetAutotest().Keyvals, ShouldNotBeNil)
				So(got.Test.GetAutotest().Keyvals["label"], ShouldEqual, "fancy-name")
			})
		})
	})
}

func TestParentIDKeyval(t *testing.T) {
	Convey("Given parent task ID", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		var params test_platform.Request_Params
		Convey("when generating a test runner request", func() {
			g := NewGenerator(inv, &params, nil, "foo-id", noDeadline)
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("the corresponding keyval is populated.", func() {
				So(got.Test, ShouldNotBeNil)
				So(got.Test.GetAutotest(), ShouldNotBeNil)
				So(got.Test.GetAutotest().Keyvals, ShouldNotBeNil)
				So(got.Test.GetAutotest().Keyvals["parent_job_id"], ShouldEqual, "foo-id")
			})
		})
	})
}

func TestBuildKeyval(t *testing.T) {
	Convey("Given a build", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		var params test_platform.Request_Params
		setBuild(&params, "foo-build")
		Convey("when generating a test runner request", func() {
			g := NewGenerator(inv, &params, nil, "", noDeadline)
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("the corresponding keyval is populated.", func() {
				So(got.Test, ShouldNotBeNil)
				So(got.Test.GetAutotest(), ShouldNotBeNil)
				So(got.Test.GetAutotest().Keyvals, ShouldNotBeNil)
				So(got.Test.GetAutotest().Keyvals["build"], ShouldEqual, "foo-build")
			})
		})
	})
}

func TestDeadline(t *testing.T) {
	Convey("Given a request that specifies a deadline", t, func() {
		ctx := context.Background()
		ts, _ := time.Parse(time.RFC3339, "2020-02-27T12:47:42Z")
		Convey("when generating a test runner request", func() {
			g := NewGenerator(basicInvocation(), &test_platform.Request_Params{}, nil, "", ts)
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("the deadline is set correctly.", func() {
				So(ptypes.TimestampString(got.Deadline), ShouldEqual, "2020-02-27T12:47:42Z")
			})
		})
	})
}

func TestNoDeadline(t *testing.T) {
	Convey("Given a request that does not specify a deadline", t, func() {
		ctx := context.Background()
		Convey("when generating a test runner request", func() {
			g := NewGenerator(basicInvocation(), &test_platform.Request_Params{}, nil, "", noDeadline)
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("the deadline should not be set.", func() {
				So(got.Deadline, ShouldBeNil)
			})
		})
	})
}

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
