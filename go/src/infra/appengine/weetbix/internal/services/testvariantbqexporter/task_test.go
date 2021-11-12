// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testvariantbqexporter

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/config"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/testutil"
	pb "infra/appengine/weetbix/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func init() {
	RegisterTaskClass()
}

func TestSchedule(t *testing.T) {
	Convey(`TestSchedule`, t, func() {
		ctx, skdr := tq.TestingContext(testutil.TestingContext(), nil)

		realm := "realm"
		cloudProject := "cloudProject"
		dataset := "dataset"
		table := "table"
		predicate := &pb.AnalyzedTestVariantPredicate{
			Status: pb.AnalyzedTestVariantStatus_FLAKY,
		}
		now := clock.Now(ctx)
		timeRange := &pb.TimeRange{
			Earliest: timestamppb.New(now.Add(-time.Hour)),
			Latest:   timestamppb.New(now),
		}
		task := &taskspb.ExportTestVariants{
			Realm:        realm,
			CloudProject: cloudProject,
			Dataset:      dataset,
			Table:        table,
			Predicate:    predicate,
			TimeRange:    timeRange,
		}
		So(Schedule(ctx, realm, cloudProject, dataset, table, predicate, timeRange), ShouldBeNil)
		So(skdr.Tasks().Payloads()[0], ShouldResembleProto, task)
	})
}

func createProjectsConfig() map[string]*config.ProjectConfig {
	return map[string]*config.ProjectConfig{
		"chromium": {
			Realms: []*config.RealmConfig{
				{
					Name: "ci",
					TestVariantAnalysis: &config.TestVariantAnalysisConfig{
						BqExports: []*config.BigQueryExport{
							{
								Table: &config.BigQueryExport_BigQueryTable{
									CloudProject: "test-hrd",
									Dataset:      "chromium",
									Table:        "flaky_test_variants_ci",
								},
							},
							{
								Table: &config.BigQueryExport_BigQueryTable{
									CloudProject: "test-hrd",
									Dataset:      "chromium",
									Table:        "flaky_test_variants_ci_copy",
								},
							},
						},
					},
				},
				{
					Name: "try",
					TestVariantAnalysis: &config.TestVariantAnalysisConfig{
						BqExports: []*config.BigQueryExport{
							{
								Table: &config.BigQueryExport_BigQueryTable{
									CloudProject: "test-hrd",
									Dataset:      "chromium",
									Table:        "flaky_test_variants_try",
								},
							},
						},
					},
				},
			},
		},
		"project_no_realms": {
			BugFilingThreshold: &config.ImpactThreshold{
				UnexpectedFailures_1D: proto.Int64(1000),
			},
		},
		"project_no_bq": {
			Realms: []*config.RealmConfig{
				{
					Name: "ci",
				},
			},
		},
	}
}

func TestScheduleTasks(t *testing.T) {
	Convey(`TestScheduleTasks`, t, func() {
		ctx, skdr := tq.TestingContext(testutil.TestingContext(), nil)
		ctx = memory.Use(ctx)
		config.SetTestProjectConfig(ctx, createProjectsConfig())

		err := ScheduleTasks(ctx)
		So(err, ShouldBeNil)
		So(len(skdr.Tasks().Payloads()), ShouldEqual, 3)
	})
}
