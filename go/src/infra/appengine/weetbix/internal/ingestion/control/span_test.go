// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package control

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/testutil"
)

func TestSpan(t *testing.T) {
	Convey(`With Spanner Test Database`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		Convey(`Read`, func() {
			entriesToCreate := []*Entry{
				NewEntry(0).WithBuildID("buildbucket-instance/1").Build(),
				NewEntry(1).WithBuildID("buildbucket-instance/2").WithProject("otherproject").Build(),
				NewEntry(2).WithBuildID("buildbucket-instance/3").WithBuildResult(nil).Build(),
				NewEntry(3).WithBuildID("buildbucket-instance/4").WithPresubmitResult(nil).Build(),
			}
			_, err := SetEntriesForTesting(ctx, entriesToCreate)
			So(err, ShouldBeNil)

			Convey(`None exist`, func() {
				buildIDs := []string{"buildbucket-instance/5"}
				results, err := Read(span.Single(ctx), testProject, buildIDs)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 1)
				So(results[0], ShouldResembleEntry, nil)
			})
			Convey(`Some exist`, func() {
				buildIDs := []string{"buildbucket-instance/3", "buildbucket-instance/4", "buildbucket-instance/2", "buildbucket-instance/1"}
				results, err := Read(span.Single(ctx), testProject, buildIDs)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 4)
				So(results[0], ShouldResembleEntry, entriesToCreate[2])
				So(results[1], ShouldResembleEntry, entriesToCreate[3])
				So(results[2], ShouldResembleEntry, nil)
				So(results[3], ShouldResembleEntry, entriesToCreate[0])
			})
		})
		Convey(`Create`, func() {
			testCreate := func(e *Entry) (time.Time, error) {
				commitTime, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
					return Create(ctx, e)
				})
				return commitTime.In(time.UTC), err
			}

			e := NewEntry(100).WithPresubmitResult(nil).Build()
			Convey(`Valid`, func() {
				// Create for first time.
				commitTime, err := testCreate(e)
				So(err, ShouldBeNil)
				e.CreationTime = commitTime
				e.LastUpdated = commitTime

				result, err := Read(span.Single(ctx), testProject, []string{e.BuildID})
				So(err, ShouldBeNil)
				So(len(result), ShouldEqual, 1)
				So(result[0], ShouldResembleEntry, e)
			})
			Convey(`With invalid Project`, func() {
				Convey(`Missing`, func() {
					e.Project = ""
					_, err := testCreate(e)
					So(err, ShouldErrLike, "project must be valid")
				})
				Convey(`Invalid`, func() {
					e.Project = "!"
					_, err := testCreate(e)
					So(err, ShouldErrLike, "project must be valid")
				})
			})
			Convey(`With missing Build ID`, func() {
				e.BuildID = ""
				_, err := testCreate(e)
				So(err, ShouldErrLike, "build ID must be specified")
			})
			Convey(`With invalid Build Result`, func() {
				Convey(`Missing host`, func() {
					e.BuildResult.Host = ""
					_, err := testCreate(e)
					So(err, ShouldErrLike, "host must be specified")
				})
				Convey(`Missing id`, func() {
					e.BuildResult.Id = 0
					_, err := testCreate(e)
					So(err, ShouldErrLike, "id must be specified")
				})
				Convey(`Missing creation time`, func() {
					e.BuildResult.CreationTime = nil
					_, err := testCreate(e)
					So(err, ShouldErrLike, "build result: creation time must be specified")
				})
			})
			Convey(`With invalid Presubmit Result`, func() {
				e = NewEntry(100).Build()
				Convey(`Missing Presubmit run ID`, func() {
					e.PresubmitResult.PresubmitRunId = nil
					_, err := testCreate(e)
					So(err, ShouldErrLike, "presubmit run ID must be specified")
				})
				Convey(`Invalid Presubmit run ID host`, func() {
					e.PresubmitResult.PresubmitRunId.System = "!"
					_, err := testCreate(e)
					So(err, ShouldErrLike, "presubmit run system must be 'luci-cv'")
				})
				Convey(`Missing Presubmit run ID system-specific ID`, func() {
					e.PresubmitResult.PresubmitRunId.Id = ""
					_, err := testCreate(e)
					So(err, ShouldErrLike, "presubmit run system-specific ID must be specified")
				})
				Convey(`Missing creation time`, func() {
					e.PresubmitResult.CreationTime = nil
					_, err := testCreate(e)
					So(err, ShouldErrLike, "presubmit result: creation time must be specified")
				})
			})
		})
		Convey(`Update`, func() {
			testUpdate := func(e *Entry) (time.Time, error) {
				commitTime, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
					return Update(ctx, e)
				})
				return commitTime.In(time.UTC), err
			}
			entriesToCreate := []*Entry{
				NewEntry(0).Build(),
			}
			_, err := SetEntriesForTesting(ctx, entriesToCreate)
			So(err, ShouldBeNil)

			e := NewEntry(1).Build()
			e.Project = entriesToCreate[0].Project
			e.BuildID = entriesToCreate[0].BuildID
			e.CreationTime = entriesToCreate[0].CreationTime

			Convey(`Valid`, func() {
				// Update.
				commitTime, err := testUpdate(e)
				So(err, ShouldBeNil)
				e.LastUpdated = commitTime

				result, err := Read(span.Single(ctx), testProject, []string{e.BuildID})
				So(err, ShouldBeNil)
				So(len(result), ShouldEqual, 1)
				So(result[0], ShouldResembleEntry, e)
			})
			Convey(`Invalid`, func() {
				// The validation implementation between Create() and Update()
				// is shared. Rather than repeat all of the same test cases,
				// we just repeat one test to ensure the method is invoked.

				Convey(`Missing Presubmit run ID`, func() {
					e.PresubmitResult.PresubmitRunId = nil
					_, err := testUpdate(e)
					So(err, ShouldErrLike, "presubmit run ID must be specified")
				})
			})
		})
		Convey(`ReadPresubmitJoinStatistics`, func() {
			Convey(`No data`, func() {
				_, err := SetEntriesForTesting(ctx, nil)
				So(err, ShouldBeNil)

				results, err := ReadPresubmitJoinStatistics(span.Single(ctx))
				So(err, ShouldBeNil)
				So(results, ShouldResemble, map[string]PresubmitJoinStatistics{})
			})
			Convey(`Data`, func() {
				reference := time.Now().Add(-1 * time.Minute)
				entriesToCreate := []*Entry{
					// Setup following data:
					// Project Alpha ("alpha") :=
					//  ]-1 hour, now]: 3 presubmit builds, 2 of which without
					//                  presubmit result.
					//                  1 non-presubmit build.
					//  ]-3 hours, -2 hour]: 2 presubmit builds, 1 of which
					//                       without build result.
					//  ]-24 hours, -23 hours]: 1 presubmit build,
					//                          with all results.
					//  ]-25 hours, -24 hours]: 1 presubmit build,
					//                          without presubmit result.
					// Project Beta ("beta") :=
					//  ]-18 hours, -17 hours]: 1 presubmit build whout build
					//                          result.
					// Project Gamma ("gamma") :=
					//  ]-25 hours, -24 hours]: 1 presubmit build,
					//                          without presubmit result.
					NewEntry(0).WithProject("alpha").WithCreationTime(reference).Build(),
					NewEntry(1).WithProject("alpha").WithCreationTime(reference).WithPresubmitResult(nil).Build(),
					NewEntry(2).WithProject("alpha").WithCreationTime(reference).WithPresubmitResult(nil).Build(),
					NewEntry(4).WithProject("alpha").WithCreationTime(reference).WithIsPresubmit(false).WithPresubmitResult(nil).Build(),
					NewEntry(5).WithProject("alpha").WithCreationTime(reference.Add(-2 * time.Hour)).Build(),
					NewEntry(6).WithProject("alpha").WithCreationTime(reference.Add(-2 * time.Hour)).WithBuildResult(nil).Build(),
					NewEntry(7).WithProject("alpha").WithCreationTime(reference.Add(-23 * time.Hour)).Build(),
					NewEntry(8).WithProject("alpha").WithCreationTime(reference.Add(-24 * time.Hour)).WithPresubmitResult(nil).Build(),
					NewEntry(9).WithProject("beta").WithCreationTime(reference.Add(-17 * time.Hour)).WithBuildResult(nil).Build(),
					NewEntry(10).WithProject("gamma").WithCreationTime(reference.Add(-24 * time.Hour)).WithPresubmitResult(nil).Build(),
				}
				_, err := SetEntriesForTesting(ctx, entriesToCreate)
				So(err, ShouldBeNil)

				results, err := ReadPresubmitJoinStatistics(span.Single(ctx))
				So(err, ShouldBeNil)

				expectedAlpha := PresubmitJoinStatistics{
					TotalBuildsByHour:             make([]int64, 24),
					AwaitingBuildByHour:           make([]int64, 24),
					AwaitingPresubmitResultByHour: make([]int64, 24),
				}
				expectedAlpha.TotalBuildsByHour[0] = 3
				expectedAlpha.AwaitingPresubmitResultByHour[0] = 2
				expectedAlpha.TotalBuildsByHour[2] = 2
				expectedAlpha.AwaitingBuildByHour[2] = 1
				expectedAlpha.TotalBuildsByHour[23] = 1
				// Only data in the last 24 hours is included, so the build
				// older than 24 hours is excluded.

				expectedBeta := PresubmitJoinStatistics{
					TotalBuildsByHour:             make([]int64, 24),
					AwaitingBuildByHour:           make([]int64, 24),
					AwaitingPresubmitResultByHour: make([]int64, 24),
				}
				expectedBeta.TotalBuildsByHour[17] = 1
				expectedBeta.AwaitingBuildByHour[17] = 1

				// Expect no entry to be returned for Project gamma
				// as all data is older than 24 hours.

				So(results, ShouldResemble, map[string]PresubmitJoinStatistics{
					"alpha": expectedAlpha,
					"beta":  expectedBeta,
				})
			})
		})
	})
}

func ShouldResembleEntry(actual interface{}, expected ...interface{}) string {
	if len(expected) != 1 {
		return fmt.Sprintf("ShouldResembleEntry expects 1 value, got %d", len(expected))
	}
	exp := expected[0]
	if exp == nil {
		return ShouldBeNil(actual)
	}

	a, ok := actual.(*Entry)
	if !ok {
		return "actual should be of type *Entry"
	}
	e, ok := exp.(*Entry)
	if !ok {
		return "expected value should be of type *Entry"
	}

	// Check equality of non-proto fields.
	a.BuildResult = nil
	a.PresubmitResult = nil
	e.BuildResult = nil
	e.PresubmitResult = nil
	if msg := ShouldResemble(a, e); msg != "" {
		return msg
	}

	// Check equality of proto fields.
	if msg := ShouldResembleProto(a.BuildResult, e.BuildResult); msg != "" {
		return msg
	}
	if msg := ShouldResembleProto(a.PresubmitResult, e.PresubmitResult); msg != "" {
		return msg
	}
	return ""
}
