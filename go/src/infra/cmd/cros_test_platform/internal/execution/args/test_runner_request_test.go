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
	"github.com/kylelemons/godebug/pretty"

	. "github.com/smartystreets/goconvey/convey"

	buildapi "go.chromium.org/chromiumos/infra/proto/go/chromite/api"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
)

func defaultTest(tests map[string]*skylab_test_runner.Request_Test) *skylab_test_runner.Request_Test {
	So(tests["original_test"], ShouldNotBeNil)
	return tests["original_test"]
}

func TestSoftwareDependencies(t *testing.T) {
	cases := []struct {
		Tag  string
		Deps []*test_platform.Request_Params_SoftwareDependency
	}{
		{
			Tag: "Chrome OS build",
			Deps: []*test_platform.Request_Params_SoftwareDependency{
				{
					Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{
						ChromeosBuild: "foo",
					},
				},
			},
		},
		{
			Tag: "RO firmware",
			Deps: []*test_platform.Request_Params_SoftwareDependency{
				{
					Dep: &test_platform.Request_Params_SoftwareDependency_RoFirmwareBuild{
						RoFirmwareBuild: "foo",
					},
				},
			},
		},
		{
			Tag: "RW firmware",
			Deps: []*test_platform.Request_Params_SoftwareDependency{
				{
					Dep: &test_platform.Request_Params_SoftwareDependency_RwFirmwareBuild{
						RwFirmwareBuild: "foo",
					},
				},
			},
		},
		{
			Tag: "lacros",
			Deps: []*test_platform.Request_Params_SoftwareDependency{
				{
					Dep: &test_platform.Request_Params_SoftwareDependency_LacrosGcsPath{
						LacrosGcsPath: "gs://some-bucket/some-build",
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Tag, func(t *testing.T) {
			g := Generator{
				Invocation: basicInvocation(),
				Params: &test_platform.Request_Params{
					SoftwareDependencies: c.Deps,
				},
			}
			got, err := g.testRunnerRequest(context.Background())
			if err != nil {
				t.Fatalf("g.testRunnerRequest() returned error: %s", err)
			}
			if diff := pretty.Compare(c.Deps, got.GetPrejob().GetSoftwareDependencies()); diff != "" {
				t.Errorf("Incorrect software dependencies, -want +got: %s", diff)
			}
		})
	}
}

func TestClientTest(t *testing.T) {
	Convey("Given a client test", t, func() {
		ctx := context.Background()
		var inv steps.EnumerationResponse_AutotestInvocation
		setExecutionEnvironment(&inv, buildapi.AutotestTest_EXECUTION_ENVIRONMENT_CLIENT)
		Convey("when generating a test runner request", func() {
			g := Generator{
				Invocation: &inv,
				Params:     &test_platform.Request_Params{},
			}
			got, err := g.testRunnerRequest(ctx)
			test := defaultTest(got.Tests)
			So(err, ShouldBeNil)
			Convey("it should be marked as such.", func() {
				So(test, ShouldNotBeNil)
				So(test.GetAutotest(), ShouldNotBeNil)
				So(test.GetAutotest().IsClientTest, ShouldEqual, true)
			})
		})
	})
}

func TestServerTest(t *testing.T) {
	Convey("Given a server test", t, func() {
		ctx := context.Background()
		var inv steps.EnumerationResponse_AutotestInvocation
		setExecutionEnvironment(&inv, buildapi.AutotestTest_EXECUTION_ENVIRONMENT_SERVER)
		Convey("when generating a test runner request", func() {
			g := Generator{
				Invocation: &inv,
				Params:     &test_platform.Request_Params{},
			}
			got, err := g.testRunnerRequest(ctx)
			test := defaultTest(got.Tests)
			So(err, ShouldBeNil)
			Convey("it should be marked as such.", func() {
				So(test, ShouldNotBeNil)
				So(test.GetAutotest(), ShouldNotBeNil)
				So(test.GetAutotest().IsClientTest, ShouldEqual, false)
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
			g := Generator{
				Invocation: &inv,
				Params:     &test_platform.Request_Params{},
			}
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
			g := Generator{
				Invocation: inv,
				Params:     &test_platform.Request_Params{},
			}
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			test := defaultTest(got.Tests)
			Convey("the test name is populated correctly.", func() {
				So(test, ShouldNotBeNil)
				So(test.GetAutotest(), ShouldNotBeNil)
				So(test.GetAutotest().Name, ShouldEqual, "foo-test")
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
			g := Generator{
				Invocation: inv,
				Params:     &test_platform.Request_Params{},
			}
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			test := defaultTest(got.Tests)
			Convey("the test args are propagated correctly.", func() {
				So(test, ShouldNotBeNil)
				So(test.GetAutotest(), ShouldNotBeNil)
				So(test.GetAutotest().TestArgs, ShouldEqual, "foo=bar baz=qux")
			})
		})
	})
}

func TestTestLevelKeyval(t *testing.T) {
	Convey("Given a keyval inside the test invocation", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		setTestKeyval(inv, "key", "test-value")
		Convey("when generating a test runner request", func() {
			g := Generator{
				Invocation: inv,
				Params:     &test_platform.Request_Params{},
			}
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			test := defaultTest(got.Tests)
			Convey("the keyval is propagated.", func() {
				So(test, ShouldNotBeNil)
				So(test.GetAutotest(), ShouldNotBeNil)
				So(test.GetAutotest().Keyvals, ShouldNotBeNil)
				So(test.GetAutotest().Keyvals["key"], ShouldEqual, "test-value")
			})
		})
	})
}

func TestRequestLevelKeyval(t *testing.T) {
	Convey("Given request-wide keyval", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		setTestKeyval(inv, "key", "test-value")
		Convey("when generating a test runner request", func() {
			g := Generator{
				Invocation: inv,
				Params:     &test_platform.Request_Params{},
			}
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			test := defaultTest(got.Tests)
			Convey("the keyval is propagated.", func() {
				So(test, ShouldNotBeNil)
				So(test.GetAutotest(), ShouldNotBeNil)
				So(test.GetAutotest().Keyvals, ShouldNotBeNil)
				So(test.GetAutotest().Keyvals["key"], ShouldEqual, "test-value")
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
			g := Generator{
				Invocation: inv,
				Params:     &params,
			}
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			test := defaultTest(got.Tests)
			Convey("the keyval from the request takes precedence.", func() {
				So(test, ShouldNotBeNil)
				So(test.GetAutotest(), ShouldNotBeNil)
				So(test.GetAutotest().Keyvals, ShouldNotBeNil)
				So(test.GetAutotest().Keyvals["ambiguous-key"], ShouldEqual, "request-value")
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
			g := Generator{
				Invocation: inv,
				Params:     &params,
			}
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			test := defaultTest(got.Tests)
			Convey("the display name is generated correctly.", func() {
				So(test, ShouldNotBeNil)
				So(test.GetAutotest(), ShouldNotBeNil)
				So(test.GetAutotest().DisplayName, ShouldEqual, "foo-build/foo-suite/foo-name")
				So(test.GetAutotest().Keyvals, ShouldNotBeNil)
				So(test.GetAutotest().Keyvals["label"], ShouldEqual, "foo-build/foo-suite/foo-name")
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
		Convey("when generating a test runner request", func() {
			g := Generator{
				Invocation: inv,
				Params:     &test_platform.Request_Params{},
			}
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			test := defaultTest(got.Tests)
			Convey("the display name is propagated correctly.", func() {
				So(test, ShouldNotBeNil)
				So(test.GetAutotest(), ShouldNotBeNil)
				So(test.GetAutotest().DisplayName, ShouldEqual, "fancy-name")
				So(test.GetAutotest().Keyvals, ShouldNotBeNil)
				So(test.GetAutotest().Keyvals["label"], ShouldEqual, "fancy-name")
			})
		})
	})
}

func TestParentIDKeyval(t *testing.T) {
	Convey("Given parent task ID", t, func() {
		ctx := context.Background()
		Convey("when generating a test runner request", func() {
			g := Generator{
				Invocation:   basicInvocation(),
				Params:       &test_platform.Request_Params{},
				ParentTaskID: "foo-id",
			}
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			test := defaultTest(got.Tests)
			Convey("the corresponding keyval is populated.", func() {
				So(test, ShouldNotBeNil)
				So(test.GetAutotest(), ShouldNotBeNil)
				So(test.GetAutotest().Keyvals, ShouldNotBeNil)
				So(test.GetAutotest().Keyvals["parent_job_id"], ShouldEqual, "foo-id")
			})
		})
	})
}

func TestBuildKeyval(t *testing.T) {
	Convey("Given a build", t, func() {
		ctx := context.Background()
		var params test_platform.Request_Params
		setBuild(&params, "foo-build")
		Convey("when generating a test runner request", func() {
			g := Generator{
				Invocation: basicInvocation(),
				Params:     &params,
			}
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			test := defaultTest(got.Tests)
			Convey("the corresponding keyval is populated.", func() {
				So(test, ShouldNotBeNil)
				So(test.GetAutotest(), ShouldNotBeNil)
				So(test.GetAutotest().Keyvals, ShouldNotBeNil)
				So(test.GetAutotest().Keyvals["build"], ShouldEqual, "foo-build")
			})
		})
	})
}

func TestDeadline(t *testing.T) {
	Convey("Given a request that specifies a deadline", t, func() {
		ctx := context.Background()
		ts, _ := time.Parse(time.RFC3339, "2020-02-27T12:47:42Z")
		Convey("when generating a test runner request", func() {
			g := Generator{
				Invocation: basicInvocation(),
				Params:     &test_platform.Request_Params{},
				Deadline:   ts,
			}
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
			g := Generator{
				Invocation: basicInvocation(),
				Params:     &test_platform.Request_Params{},
			}
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("the deadline should not be set.", func() {
				So(got.Deadline, ShouldBeNil)
			})
		})
	})
}

func TestParentUID(t *testing.T) {
	Convey("Given a parent UID", t, func() {
		ctx := context.Background()
		Convey("when generating a test runner request", func() {
			g := Generator{
				Invocation:       basicInvocation(),
				Params:           &test_platform.Request_Params{},
				ParentTaskID:     "foo-id",
				ParentRequestUID: "foo-uid",
			}
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("the parent UID is propagated correctly.", func() {
				So(got.ParentRequestUid, ShouldEqual, "foo-uid")
			})
		})
	})
}

func TestParentBuildID(t *testing.T) {
	Convey("Given a parent UID", t, func() {
		ctx := context.Background()
		Convey("when generating a test runner request", func() {
			g := Generator{
				Invocation:    basicInvocation(),
				Params:        &test_platform.Request_Params{},
				ParentBuildID: 43,
			}
			got, err := g.testRunnerRequest(ctx)
			So(err, ShouldBeNil)
			Convey("the parent build ID is propagated correctly.", func() {
				So(got.ParentBuildId, ShouldEqual, 43)
			})
		})
	})
}
