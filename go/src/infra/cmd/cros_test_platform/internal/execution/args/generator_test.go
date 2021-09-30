// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package args contains the logic for assembling all data required for
// creating an individual task request.
package args

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	bbpb "go.chromium.org/luci/buildbucket/proto"
)

func TestDisplayNameTagsForUnamedRequest(t *testing.T) {
	Convey("Given a request does not specify a display name", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		setTestName(inv, "foo-name")
		var params test_platform.Request_Params
		var dummyWorkerConfig = &config.Config_SkylabWorker{}
		setBuild(&params, "foo-build")
		setRequestKeyval(&params, "suite", "foo-suite")
		setRequestMaximumDuration(&params, 1000)
		Convey("when generating a test runner request's args", func() {
			g := Generator{
				Invocation:       inv,
				Params:           &params,
				WorkerConfig:     dummyWorkerConfig,
				ParentRequestUID: "TestPlanRuns/12345678/foo",
			}
			got, err := g.GenerateArgs(ctx)
			So(err, ShouldBeNil)
			Convey("the display name tag is generated correctly.", func() {
				So(got.SwarmingTags, ShouldContain, "display_name:foo-build/foo-suite/foo-name")
				So(got.ParentRequestUID, ShouldEqual, "TestPlanRuns/12345678/foo")
			})
		})
	})
}

func TestInventoryLabels(t *testing.T) {
	Convey("Given a request with board and model info", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		setTestName(inv, "foo-name")
		var params test_platform.Request_Params
		var dummyWorkerConfig = &config.Config_SkylabWorker{}
		setRequestMaximumDuration(&params, 1000)
		setPrimayDeviceBoard(&params, "coral")
		setPrimayDeviceModel(&params, "babytiger")
		Convey("when generating a test runner request's args", func() {
			g := Generator{
				Invocation:   inv,
				Params:       &params,
				WorkerConfig: dummyWorkerConfig,
			}
			got, err := g.GenerateArgs(ctx)
			So(err, ShouldBeNil)
			Convey("the SchedulableLabels is generated correctly", func() {
				So(*got.SchedulableLabels.Board, ShouldEqual, "coral")
				So(*got.SchedulableLabels.Model, ShouldEqual, "babytiger")
				So(len(got.SecondaryDevicesLabels), ShouldEqual, 0)
			})
		})
	})
}

func TestSecondaryDevicesLabels(t *testing.T) {
	Convey("Given a request with secondary devices", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		setTestName(inv, "foo-name")
		var params test_platform.Request_Params
		var dummyWorkerConfig = &config.Config_SkylabWorker{}
		setRequestMaximumDuration(&params, 1000)
		setSecondaryDevice(&params, "nami", "", "")
		setSecondaryDevice(&params, "coral", "babytiger", "")
		Convey("when generating a test runner request's args", func() {
			g := Generator{
				Invocation:   inv,
				Params:       &params,
				WorkerConfig: dummyWorkerConfig,
			}
			got, err := g.GenerateArgs(ctx)
			So(err, ShouldBeNil)
			Convey("the SecondaryDevicesLabels is generated correctly", func() {
				So(len(got.SecondaryDevicesLabels), ShouldEqual, 2)
				So(*got.SecondaryDevicesLabels[0].Board, ShouldEqual, "nami")
				So(*got.SecondaryDevicesLabels[0].Model, ShouldEqual, "")
				So(*got.SecondaryDevicesLabels[1].Board, ShouldEqual, "coral")
				So(*got.SecondaryDevicesLabels[1].Model, ShouldEqual, "babytiger")
			})
		})
	})
}

func TestExperiments(t *testing.T) {
	Convey("Given a request with experiments", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		setTestName(inv, "foo-name")
		var params test_platform.Request_Params
		var dummyWorkerConfig = &config.Config_SkylabWorker{}
		setRequestMaximumDuration(&params, 1000)
		Convey("when generating a test runner request's args", func() {
			g := Generator{
				Invocation:   inv,
				Params:       &params,
				WorkerConfig: dummyWorkerConfig,
				Experiments:  []string{"exp1", "exp2"},
			}
			got, err := g.GenerateArgs(ctx)
			So(err, ShouldBeNil)
			Convey("the experiments field is propogated correctly", func() {
				So(len(got.Experiments), ShouldEqual, 2)
				So(got.Experiments, ShouldResemble, []string{"exp1", "exp2"})
			})
		})
	})
}

func TestGerritChanges(t *testing.T) {
	Convey("Given Gerrit Changes", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		setTestName(inv, "foo-name")
		var params test_platform.Request_Params
		var dummyWorkerConfig = &config.Config_SkylabWorker{}
		setRequestMaximumDuration(&params, 1000)
		Convey("when generating a test runner request's args", func() {
			gc := &bbpb.GerritChange{
				Host:     "a",
				Project:  "b",
				Change:   123,
				Patchset: 1,
			}
			g := Generator{
				Invocation:    inv,
				Params:        &params,
				WorkerConfig:  dummyWorkerConfig,
				GerritChanges: []*bbpb.GerritChange{gc},
			}
			got, err := g.GenerateArgs(ctx)
			So(err, ShouldBeNil)
			Convey("the GerritChanges are added correctly", func() {
				So(len(got.GerritChanges), ShouldEqual, 1)
				So(got.GerritChanges, ShouldResemble, []*bbpb.GerritChange{gc})
			})
		})
	})
}
