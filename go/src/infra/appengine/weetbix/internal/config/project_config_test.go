// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"testing"
	"time"

	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/cfgclient"
	cfgmem "go.chromium.org/luci/config/impl/memory"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server/caching"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/common/clock/testclock"
	. "go.chromium.org/luci/common/testing/assertions"
)

var textPBMultiline = prototext.MarshalOptions{
	Multiline: true,
}

func createImpactThreshold() *ImpactThreshold {
	return &ImpactThreshold{
		UnexpectedFailures_1D: proto.Int64(1000),
	}
}

func createProjectConfig() *ProjectConfig {
	return &ProjectConfig{
		Monorail: &MonorailProject{
			Project:         "chromium",
			PriorityFieldId: 10,
			Priorities: []*MonorailPriority{
				{
					Priority:  "0",
					Threshold: createImpactThreshold(),
				},
			},
		},
		BugFilingThreshold: createImpactThreshold(),
	}
}

func TestProjectConfig(t *testing.T) {
	t.Parallel()

	Convey("SetTestProjectConfig updates context config", t, func() {
		projectA := createProjectConfig()
		configs := make(map[string]*ProjectConfig)
		configs["a"] = projectA

		ctx := memory.Use(context.Background())
		SetTestProjectConfig(ctx, configs)

		cfg, err := Projects(ctx)
		So(err, ShouldBeNil)
		So(len(cfg), ShouldEqual, 1)
		So(cfg["a"], ShouldResembleProto, projectA)
	})

	Convey("With mocks", t, func() {
		projectA := createProjectConfig()
		projectB := createProjectConfig()
		projectB.Monorail.PriorityFieldId = 1

		configs := map[config.Set]cfgmem.Files{
			"projects/a": {"${appid}.cfg": textPBMultiline.Format(projectA)},
			"projects/b": {"${appid}.cfg": textPBMultiline.Format(projectB)},
		}

		ctx := memory.Use(context.Background())
		ctx, tc := testclock.UseTime(ctx, testclock.TestTimeUTC)
		ctx = cfgclient.Use(ctx, cfgmem.New(configs))
		ctx = caching.WithEmptyProcessCache(ctx)

		Convey("Update works", func() {

			// Initial update.
			err := updateProjects(ctx)
			So(err, ShouldBeNil)
			datastore.GetTestable(ctx).CatchupIndexes()

			// Get works.
			projects, err := Projects(ctx)
			So(err, ShouldBeNil)
			So(len(projects), ShouldEqual, 2)
			So(projects["a"], ShouldResembleProto, projectA)
			So(projects["b"], ShouldResembleProto, projectB)

			// Noop update.
			err = updateProjects(ctx)
			So(err, ShouldBeNil)
			datastore.GetTestable(ctx).CatchupIndexes()

			// Real update.
			projectC := createProjectConfig()
			newProjectB := createProjectConfig()
			newProjectB.Monorail.PriorityFieldId = 2
			delete(configs, "projects/a")
			configs["projects/b"]["${appid}.cfg"] = textPBMultiline.Format(newProjectB)
			configs["projects/c"] = cfgmem.Files{
				"${appid}.cfg": textPBMultiline.Format(projectC),
			}
			err = updateProjects(ctx)
			So(err, ShouldBeNil)
			datastore.GetTestable(ctx).CatchupIndexes()

			// Fetch returns the new value right away.
			projects, err = fetchProjects(ctx)
			So(err, ShouldBeNil)
			So(len(projects), ShouldEqual, 2)
			So(projects["b"], ShouldResembleProto, newProjectB)
			So(projects["c"], ShouldResembleProto, projectC)

			// Get still uses in-memory cached copy.
			projects, err = Projects(ctx)
			So(err, ShouldBeNil)
			So(len(projects), ShouldEqual, 2)
			So(projects["a"], ShouldResembleProto, projectA)
			So(projects["b"], ShouldResembleProto, projectB)

			// Time passes, in-memory cached copy expires.
			tc.Add(2 * time.Minute)

			// Get returns the new value now too.
			projects, err = Projects(ctx)
			So(err, ShouldBeNil)
			So(len(projects), ShouldEqual, 2)
			So(projects["b"], ShouldResembleProto, newProjectB)
			So(projects["c"], ShouldResembleProto, projectC)
		})

		Convey("Validation works", func() {
			configs["projects/b"]["${appid}.cfg"] = `bad data`
			err := updateProjects(ctx)
			datastore.GetTestable(ctx).CatchupIndexes()
			So(err, ShouldErrLike, "validation errors")

			// Validation for project A passed and project is
			// available, validation for project B failed
			// as is not available.
			projects, err := Projects(ctx)
			So(err, ShouldBeNil)
			So(len(projects), ShouldEqual, 1)
			So(projects["a"], ShouldResembleProto, projectA)
		})

		Convey("Update retains existing config if new config is invalid", func() {
			// Initial update.
			err := updateProjects(ctx)
			So(err, ShouldBeNil)
			datastore.GetTestable(ctx).CatchupIndexes()

			// Get works.
			projects, err := Projects(ctx)
			So(err, ShouldBeNil)
			So(len(projects), ShouldEqual, 2)
			So(projects["a"], ShouldResembleProto, projectA)
			So(projects["b"], ShouldResembleProto, projectB)

			// Attempt to update with an invalid config for project B.
			newProjectA := createProjectConfig()
			newProjectA.Monorail.Project = "new-project-a"
			newProjectB := createProjectConfig()
			newProjectB.Monorail.Project = ""
			configs["projects/a"]["${appid}.cfg"] = textPBMultiline.Format(newProjectA)
			configs["projects/b"]["${appid}.cfg"] = textPBMultiline.Format(newProjectB)
			err = updateProjects(ctx)
			So(err, ShouldErrLike, "validation errors")
			datastore.GetTestable(ctx).CatchupIndexes()

			// Time passes, in-memory cached copy expires.
			tc.Add(2 * time.Minute)

			// Get returns the new configuration A and the old
			// configuration for B. This ensures an attempt to push an invalid
			// config does not result in a service outage for that project.
			projects, err = Projects(ctx)
			So(err, ShouldBeNil)
			So(len(projects), ShouldEqual, 2)
			So(projects["a"], ShouldResembleProto, newProjectA)
			So(projects["b"], ShouldResembleProto, projectB)
		})
	})
}
