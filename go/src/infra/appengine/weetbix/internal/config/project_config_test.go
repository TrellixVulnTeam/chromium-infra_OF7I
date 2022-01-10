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
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	configpb "infra/appengine/weetbix/internal/config/proto"
	pb "infra/appengine/weetbix/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	. "go.chromium.org/luci/common/testing/assertions"
)

var textPBMultiline = prototext.MarshalOptions{
	Multiline: true,
}

func createProjectConfig() *configpb.ProjectConfig {
	return &configpb.ProjectConfig{
		Monorail: &configpb.MonorailProject{
			Project:         "chromium",
			PriorityFieldId: 10,
			Priorities: []*configpb.MonorailPriority{
				{
					Priority: "0",
					Threshold: &configpb.ImpactThreshold{
						TestResultsFailed: &configpb.MetricThreshold{
							OneDay: proto.Int64(1500),
						},
					},
				},
				{
					Priority: "1",
					Threshold: &configpb.ImpactThreshold{
						TestResultsFailed: &configpb.MetricThreshold{
							OneDay: proto.Int64(500),
						},
					},
				},
			},
		},
		BugFilingThreshold: &configpb.ImpactThreshold{
			TestResultsFailed: &configpb.MetricThreshold{
				OneDay: proto.Int64(1000),
			},
		},
		Realms: []*configpb.RealmConfig{
			{
				Name: "ci",
				TestVariantAnalysis: &configpb.TestVariantAnalysisConfig{
					UpdateTestVariantTask: &configpb.UpdateTestVariantTask{
						UpdateTestVariantTaskInterval:   durationpb.New(time.Hour),
						TestVariantStatusUpdateDuration: durationpb.New(6 * time.Hour),
					},
					BqExports: []*configpb.BigQueryExport{
						{
							Table: &configpb.BigQueryExport_BigQueryTable{
								CloudProject: "test-hrd",
								Dataset:      "chromium",
								Table:        "flaky_test_variants",
							},
							Predicate: &pb.AnalyzedTestVariantPredicate{},
						},
					},
				},
			},
		},
		Clustering: &configpb.Clustering{
			TestNameRules: []*configpb.TestNameClusteringRule{
				{
					Name:         "Google Test (Value-parameterized)",
					Pattern:      `^ninja:(?P<target>[\w/]+:\w+)/` + `(\w+/)?(?P<suite>\w+)\.(?P<case>\w+)/\w+$`,
					LikeTemplate: `ninja:${target}/%${suite}.${case}%`,
				},
				{
					Name:         "Google Test (Type-parameterized)",
					Pattern:      `^ninja:(?P<target>[\w/]+:\w+)/` + `(\w+/)?(?P<suite>\w+)/\w+\.(?P<case>\w+)$`,
					LikeTemplate: `ninja:${target}/%${suite}/%.${case}`,
				},
			},
		},
	}
}

func TestProjectConfig(t *testing.T) {
	t.Parallel()

	Convey("SetTestProjectConfig updates context config", t, func() {
		projectA := createProjectConfig()
		projectA.LastUpdated = timestamppb.New(time.Now())
		configs := make(map[string]*configpb.ProjectConfig)
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
			creationTime := clock.Now(ctx)
			err := updateProjects(ctx)
			So(err, ShouldBeNil)
			datastore.GetTestable(ctx).CatchupIndexes()

			// Get works.
			projects, err := Projects(ctx)
			So(err, ShouldBeNil)
			So(len(projects), ShouldEqual, 2)
			So(projects["a"], ShouldResembleProto, withLastUpdated(projectA, creationTime))
			So(projects["b"], ShouldResembleProto, withLastUpdated(projectB, creationTime))

			tc.Add(1 * time.Second)

			// Noop update.
			err = updateProjects(ctx)
			So(err, ShouldBeNil)
			datastore.GetTestable(ctx).CatchupIndexes()

			tc.Add(1 * time.Second)

			// Real update.
			projectC := createProjectConfig()
			newProjectB := createProjectConfig()
			newProjectB.Monorail.PriorityFieldId = 2
			delete(configs, "projects/a")
			configs["projects/b"]["${appid}.cfg"] = textPBMultiline.Format(newProjectB)
			configs["projects/c"] = cfgmem.Files{
				"${appid}.cfg": textPBMultiline.Format(projectC),
			}
			updateTime := clock.Now(ctx)
			err = updateProjects(ctx)
			So(err, ShouldBeNil)
			datastore.GetTestable(ctx).CatchupIndexes()

			// Fetch returns the new value right away.
			projects, err = fetchProjects(ctx)
			So(err, ShouldBeNil)
			So(len(projects), ShouldEqual, 2)
			So(projects["b"], ShouldResembleProto, withLastUpdated(newProjectB, updateTime))
			So(projects["c"], ShouldResembleProto, withLastUpdated(projectC, updateTime))

			// Get still uses in-memory cached copy.
			projects, err = Projects(ctx)
			So(err, ShouldBeNil)
			So(len(projects), ShouldEqual, 2)
			So(projects["a"], ShouldResembleProto, withLastUpdated(projectA, creationTime))
			So(projects["b"], ShouldResembleProto, withLastUpdated(projectB, creationTime))

			Convey("Expedited cache eviction", func() {
				projectB, err = ProjectWithMinimumVersion(ctx, "b", updateTime)
				So(err, ShouldBeNil)
				So(projectB, ShouldResembleProto, withLastUpdated(newProjectB, updateTime))
			})
			Convey("Natural cache eviction", func() {
				// Time passes, in-memory cached copy expires.
				tc.Add(2 * time.Minute)

				// Get returns the new value now too.
				projects, err = Projects(ctx)
				So(err, ShouldBeNil)
				So(len(projects), ShouldEqual, 2)
				So(projects["b"], ShouldResembleProto, withLastUpdated(newProjectB, updateTime))
				So(projects["c"], ShouldResembleProto, withLastUpdated(projectC, updateTime))

				// Time passes, in-memory cached copy expires.
				tc.Add(2 * time.Minute)

				// Get returns the same value.
				projects, err = Projects(ctx)
				So(err, ShouldBeNil)
				So(len(projects), ShouldEqual, 2)
				So(projects["b"], ShouldResembleProto, withLastUpdated(newProjectB, updateTime))
				So(projects["c"], ShouldResembleProto, withLastUpdated(projectC, updateTime))
			})
		})

		Convey("Validation works", func() {
			configs["projects/b"]["${appid}.cfg"] = `bad data`
			creationTime := clock.Now(ctx)
			err := updateProjects(ctx)
			datastore.GetTestable(ctx).CatchupIndexes()
			So(err, ShouldErrLike, "validation errors")

			// Validation for project A passed and project is
			// available, validation for project B failed
			// as is not available.
			projects, err := Projects(ctx)
			So(err, ShouldBeNil)
			So(len(projects), ShouldEqual, 1)
			So(projects["a"], ShouldResembleProto, withLastUpdated(projectA, creationTime))
		})

		Convey("Update retains existing config if new config is invalid", func() {
			// Initial update.
			creationTime := clock.Now(ctx)
			err := updateProjects(ctx)
			So(err, ShouldBeNil)
			datastore.GetTestable(ctx).CatchupIndexes()

			// Get works.
			projects, err := Projects(ctx)
			So(err, ShouldBeNil)
			So(len(projects), ShouldEqual, 2)
			So(projects["a"], ShouldResembleProto, withLastUpdated(projectA, creationTime))
			So(projects["b"], ShouldResembleProto, withLastUpdated(projectB, creationTime))

			tc.Add(1 * time.Second)

			// Attempt to update with an invalid config for project B.
			newProjectA := createProjectConfig()
			newProjectA.Monorail.Project = "new-project-a"
			newProjectB := createProjectConfig()
			newProjectB.Monorail.Project = ""
			configs["projects/a"]["${appid}.cfg"] = textPBMultiline.Format(newProjectA)
			configs["projects/b"]["${appid}.cfg"] = textPBMultiline.Format(newProjectB)
			updateTime := clock.Now(ctx)
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
			So(projects["a"], ShouldResembleProto, withLastUpdated(newProjectA, updateTime))
			So(projects["b"], ShouldResembleProto, withLastUpdated(projectB, creationTime))
		})
	})
}

// withLastUpdated returns a copy of the given ProjectConfig with the
// specified LastUpdated time set.
func withLastUpdated(cfg *configpb.ProjectConfig, lastUpdated time.Time) *configpb.ProjectConfig {
	result := proto.Clone(cfg).(*configpb.ProjectConfig)
	result.LastUpdated = timestamppb.New(lastUpdated)
	return result
}

func TestProject(t *testing.T) {
	t.Parallel()

	Convey("Project", t, func() {
		pjChromium := createProjectConfig()
		configs := map[string]*configpb.ProjectConfig{
			"chromium": pjChromium,
		}

		ctx := memory.Use(context.Background())
		SetTestProjectConfig(ctx, configs)

		Convey("success", func() {
			pj, err := Project(ctx, "chromium")
			So(err, ShouldBeNil)
			So(pj, ShouldResembleProto, pjChromium)
		})

		Convey("not found", func() {
			pj, err := Project(ctx, "random")
			So(err, ShouldErrLike, "no config found for project random")
			So(pj, ShouldBeNil)
		})
	})
}

func TestRealm(t *testing.T) {
	t.Parallel()

	Convey("Realm", t, func() {
		pj := createProjectConfig()
		configs := map[string]*configpb.ProjectConfig{
			"chromium": pj,
		}

		ctx := memory.Use(context.Background())
		SetTestProjectConfig(ctx, configs)

		Convey("success", func() {
			rj, err := Realm(ctx, "chromium:ci")
			So(err, ShouldBeNil)
			So(rj, ShouldResembleProto, pj.Realms[0])
		})

		Convey("not found", func() {
			rj, err := Realm(ctx, "chromium:random")
			So(err, ShouldErrLike, "no config found for realm chromium:random")
			So(rj, ShouldBeNil)
		})
	})
}
