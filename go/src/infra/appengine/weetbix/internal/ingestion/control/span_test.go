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
				NewEntry(2).WithBuildID("buildbucket-instance/2").WithBuildResult(nil).Build(),
				NewEntry(3).WithBuildID("buildbucket-instance/3").WithPresubmitResult(nil).Build(),
			}
			_, err := SetEntriesForTesting(ctx, entriesToCreate)
			So(err, ShouldBeNil)

			Convey(`None exist`, func() {
				buildIDs := []string{"buildbucket-instance/4"}
				results, err := Read(span.Single(ctx), buildIDs)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 1)
				So(results[0], ShouldResembleEntry, nil)
			})
			Convey(`Some exist`, func() {
				buildIDs := []string{
					"buildbucket-instance/3",
					"buildbucket-instance/4",
					"buildbucket-instance/2",
					"buildbucket-instance/1",
				}
				results, err := Read(span.Single(ctx), buildIDs)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 4)
				So(results[0], ShouldResembleEntry, entriesToCreate[2])
				So(results[1], ShouldResembleEntry, nil)
				So(results[2], ShouldResembleEntry, entriesToCreate[1])
				So(results[3], ShouldResembleEntry, entriesToCreate[0])
			})
		})
		Convey(`SetBuildResult`, func() {
			testSetBuildResult := func(e *Entry) (time.Time, error) {
				commitTime, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
					return SetBuildResult(ctx, e)
				})
				return commitTime.In(time.UTC), err
			}

			entriesToCreate := []*Entry{
				NewEntry(0).Build(),
			}
			_, err := SetEntriesForTesting(ctx, entriesToCreate)
			So(err, ShouldBeNil)

			e := NewEntry(1).Build()

			Convey(`Valid`, func() {
				Convey(`Create`, func() {
					commitTime, err := testSetBuildResult(e)
					So(err, ShouldBeNil)
					e.BuildJoinedTime = commitTime
					e.LastUpdated = commitTime

					// SetBuildResult should not have set these fields.
					e.PresubmitProject = ""
					e.PresubmitResult = nil
					e.PresubmitJoinedTime = time.Time{}

					result, err := Read(span.Single(ctx), []string{e.BuildID})
					So(err, ShouldBeNil)
					So(len(result), ShouldEqual, 1)
					So(result[0], ShouldResembleEntry, e)
				})
				Convey(`Update`, func() {
					// Update the existing entry.
					e.BuildID = entriesToCreate[0].BuildID

					commitTime, err := testSetBuildResult(e)
					So(err, ShouldBeNil)
					e.BuildJoinedTime = commitTime
					e.LastUpdated = commitTime

					// SetBuildResult should not update these fields.
					e.PresubmitProject = entriesToCreate[0].PresubmitProject
					e.PresubmitResult = entriesToCreate[0].PresubmitResult
					e.PresubmitJoinedTime = entriesToCreate[0].PresubmitJoinedTime

					result, err := Read(span.Single(ctx), []string{e.BuildID})
					So(err, ShouldBeNil)
					So(len(result), ShouldEqual, 1)
					So(result[0], ShouldResembleEntry, e)
				})
			})
			Convey(`With invalid Build Project`, func() {
				Convey(`Missing`, func() {
					e.BuildProject = ""
					_, err := testSetBuildResult(e)
					So(err, ShouldErrLike, "build project must be valid")
				})
				Convey(`Invalid`, func() {
					e.BuildProject = "!"
					_, err := testSetBuildResult(e)
					So(err, ShouldErrLike, "build project must be valid")
				})
			})
			Convey(`With missing Build ID`, func() {
				e.BuildID = ""
				_, err := testSetBuildResult(e)
				So(err, ShouldErrLike, "build ID must be specified")
			})
			Convey(`With invalid Build Result`, func() {
				Convey(`Missing host`, func() {
					e.BuildResult.Host = ""
					_, err := testSetBuildResult(e)
					So(err, ShouldErrLike, "host must be specified")
				})
				Convey(`Missing id`, func() {
					e.BuildResult.Id = 0
					_, err := testSetBuildResult(e)
					So(err, ShouldErrLike, "id must be specified")
				})
				Convey(`Missing creation time`, func() {
					e.BuildResult.CreationTime = nil
					_, err := testSetBuildResult(e)
					So(err, ShouldErrLike, "build result: creation time must be specified")
				})
			})
		})
		Convey(`SetPresubmitResult`, func() {
			testSetPresubmitResult := func(e *Entry) (time.Time, error) {
				commitTime, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
					return SetPresubmitResult(ctx, e)
				})
				return commitTime.In(time.UTC), err
			}
			entriesToCreate := []*Entry{
				NewEntry(0).Build(),
			}
			_, err := SetEntriesForTesting(ctx, entriesToCreate)
			So(err, ShouldBeNil)

			e := NewEntry(1).Build()

			Convey(`Valid`, func() {
				Convey(`Create`, func() {
					commitTime, err := testSetPresubmitResult(e)
					So(err, ShouldBeNil)
					e.PresubmitJoinedTime = commitTime
					e.LastUpdated = commitTime

					// SetPresubmitResult should not have set these fields.
					e.BuildProject = ""
					e.BuildResult = nil
					e.BuildJoinedTime = time.Time{}

					result, err := Read(span.Single(ctx), []string{e.BuildID})
					So(err, ShouldBeNil)
					So(len(result), ShouldEqual, 1)
					So(result[0], ShouldResembleEntry, e)
				})
				Convey(`Update`, func() {
					// Update the existing entry.
					e.BuildID = entriesToCreate[0].BuildID

					commitTime, err := testSetPresubmitResult(e)
					So(err, ShouldBeNil)
					e.PresubmitJoinedTime = commitTime
					e.LastUpdated = commitTime

					// SetPresubmitResult should not update these fields.
					e.BuildProject = entriesToCreate[0].BuildProject
					e.BuildResult = entriesToCreate[0].BuildResult
					e.BuildJoinedTime = entriesToCreate[0].BuildJoinedTime

					result, err := Read(span.Single(ctx), []string{e.BuildID})
					So(err, ShouldBeNil)
					So(len(result), ShouldEqual, 1)
					So(result[0], ShouldResembleEntry, e)
				})
			})
			Convey(`Invalid`, func() {
				Convey(`With invalid Presubmit Project`, func() {
					Convey(`Missing`, func() {
						e.PresubmitProject = ""
						_, err := testSetPresubmitResult(e)
						So(err, ShouldErrLike, "presubmit project must be valid")
					})
					Convey(`Invalid`, func() {
						e.PresubmitProject = "!"
						_, err := testSetPresubmitResult(e)
						So(err, ShouldErrLike, "presubmit project must be valid")
					})
				})
				Convey(`With missing Build ID`, func() {
					e.BuildID = ""
					_, err := testSetPresubmitResult(e)
					So(err, ShouldErrLike, "build ID must be specified")
				})
				Convey(`Missing Presubmit run ID`, func() {
					e.PresubmitResult.PresubmitRunId = nil
					_, err := testSetPresubmitResult(e)
					So(err, ShouldErrLike, "presubmit run ID must be specified")
				})
				Convey(`With invalid Presubmit Result`, func() {
					e = NewEntry(100).Build()
					Convey(`Missing Presubmit run ID`, func() {
						e.PresubmitResult.PresubmitRunId = nil
						_, err := testSetPresubmitResult(e)
						So(err, ShouldErrLike, "presubmit run ID must be specified")
					})
					Convey(`Invalid Presubmit run ID host`, func() {
						e.PresubmitResult.PresubmitRunId.System = "!"
						_, err := testSetPresubmitResult(e)
						So(err, ShouldErrLike, "presubmit run system must be 'luci-cv'")
					})
					Convey(`Missing Presubmit run ID system-specific ID`, func() {
						e.PresubmitResult.PresubmitRunId.Id = ""
						_, err := testSetPresubmitResult(e)
						So(err, ShouldErrLike, "presubmit run system-specific ID must be specified")
					})
					Convey(`Missing creation time`, func() {
						e.PresubmitResult.CreationTime = nil
						_, err := testSetPresubmitResult(e)
						So(err, ShouldErrLike, "presubmit result: creation time must be specified")
					})
				})
			})
		})
		Convey(`ReadPresubmitRunJoinStatistics`, func() {
			Convey(`No data`, func() {
				_, err := SetEntriesForTesting(ctx, nil)
				So(err, ShouldBeNil)

				results, err := ReadPresubmitRunJoinStatistics(span.Single(ctx))
				So(err, ShouldBeNil)
				So(results, ShouldResemble, map[string]JoinStatistics{})
			})
			Convey(`Data`, func() {
				reference := time.Now().Add(-1 * time.Minute)
				entriesToCreate := []*Entry{
					// Setup following data:
					// Project Alpha ("alpha") :=
					//  ]-1 hour, now]: 4 presubmit builds, 2 of which without
					//                  presubmit result, 1 of which without
					//                  build result.
					//                  1 non-presubmit build.
					//  ]-36 hours, -35 hours]: 1 presubmit build,
					//                          with all results.
					//  ]-37 hours, -36 hours]: 1 presubmit build,
					//                          with all results
					//                         (should be ignored).
					// Project Beta ("beta") :=
					//  ]-37 hours, -36 hours]: 1 presubmit build,
					//                          without presubmit result.
					NewEntry(0).WithBuildProject("alpha").WithBuildJoinedTime(reference).Build(),
					NewEntry(1).WithBuildProject("alpha").WithBuildJoinedTime(reference).WithPresubmitResult(nil).Build(),
					NewEntry(2).WithBuildProject("alpha").WithBuildJoinedTime(reference).WithPresubmitResult(nil).Build(),
					NewEntry(3).WithPresubmitProject("alpha").WithPresubmitJoinedTime(reference).WithBuildResult(nil).Build(),
					NewEntry(4).WithBuildProject("alpha").WithBuildJoinedTime(reference).WithIsPresubmit(false).WithPresubmitResult(nil).Build(),
					NewEntry(5).WithBuildProject("alpha").WithBuildJoinedTime(reference.Add(-35 * time.Hour)).Build(),
					NewEntry(6).WithBuildProject("alpha").WithBuildJoinedTime(reference.Add(-36 * time.Hour)).Build(),
					NewEntry(7).WithBuildProject("beta").WithBuildJoinedTime(reference.Add(-36 * time.Hour)).WithPresubmitResult(nil).Build(),
				}
				_, err := SetEntriesForTesting(ctx, entriesToCreate)
				So(err, ShouldBeNil)

				results, err := ReadPresubmitRunJoinStatistics(span.Single(ctx))
				So(err, ShouldBeNil)

				expectedAlpha := JoinStatistics{
					TotalByHour:  make([]int64, 36),
					JoinedByHour: make([]int64, 36),
				}
				expectedAlpha.TotalByHour[0] = 3
				expectedAlpha.JoinedByHour[0] = 1
				expectedAlpha.TotalByHour[35] = 1
				expectedAlpha.JoinedByHour[35] = 1
				// Only data in the last 36 hours is included, so the build
				// older than 36 hours is excluded.

				// Expect no entry to be returned for Project beta
				// as all data is older than 36 hours.

				So(results, ShouldResemble, map[string]JoinStatistics{
					"alpha": expectedAlpha,
				})
			})
		})
		Convey(`ReadBuildJoinStatistics`, func() {
			Convey(`No data`, func() {
				_, err := SetEntriesForTesting(ctx, nil)
				So(err, ShouldBeNil)

				results, err := ReadBuildJoinStatistics(span.Single(ctx))
				So(err, ShouldBeNil)
				So(results, ShouldResemble, map[string]JoinStatistics{})
			})
			Convey(`Data`, func() {
				reference := time.Now().Add(-1 * time.Minute)
				entriesToCreate := []*Entry{
					// Setup following data:
					// Project Alpha ("alpha") :=
					//  ]-1 hour, now]: 4 presubmit builds, 2 of which without
					//                  build result, 1 of which without
					//                  presubmit result.
					//                  1 non-presubmit build.
					//  ]-36 hours, -35 hours]: 1 presubmit build,
					//                          with all results.
					//  ]-37 hours, -36 hours]: 1 presubmit build,
					//                          with all results
					//                          (should be ignored).
					// Project Beta ("beta") :=
					//  ]-37 hours, -36 hours]: 1 presubmit build,
					//                          without build result.
					NewEntry(0).WithPresubmitProject("alpha").WithPresubmitJoinedTime(reference).Build(),
					NewEntry(1).WithPresubmitProject("alpha").WithPresubmitJoinedTime(reference).WithBuildResult(nil).Build(),
					NewEntry(2).WithPresubmitProject("alpha").WithPresubmitJoinedTime(reference).WithBuildResult(nil).Build(),
					NewEntry(3).WithBuildProject("alpha").WithBuildJoinedTime(reference).WithPresubmitResult(nil).Build(),
					NewEntry(4).WithPresubmitProject("alpha").WithPresubmitJoinedTime(reference).WithIsPresubmit(false).WithBuildResult(nil).Build(),
					NewEntry(5).WithPresubmitProject("alpha").WithPresubmitJoinedTime(reference.Add(-35 * time.Hour)).Build(),
					NewEntry(6).WithPresubmitProject("alpha").WithPresubmitJoinedTime(reference.Add(-36 * time.Hour)).Build(),
					NewEntry(7).WithPresubmitProject("beta").WithPresubmitJoinedTime(reference.Add(-36 * time.Hour)).WithBuildResult(nil).Build(),
				}
				_, err := SetEntriesForTesting(ctx, entriesToCreate)
				So(err, ShouldBeNil)

				results, err := ReadBuildJoinStatistics(span.Single(ctx))
				So(err, ShouldBeNil)

				expectedAlpha := JoinStatistics{
					TotalByHour:  make([]int64, 36),
					JoinedByHour: make([]int64, 36),
				}
				expectedAlpha.TotalByHour[0] = 3
				expectedAlpha.JoinedByHour[0] = 1
				expectedAlpha.TotalByHour[35] = 1
				expectedAlpha.JoinedByHour[35] = 1
				// Only data in the last 36 hours is included, so the build
				// older than 36 hours is excluded.

				// Expect no entry to be returned for Project beta
				// as all data is older than 36 hours.

				So(results, ShouldResemble, map[string]JoinStatistics{
					"alpha": expectedAlpha,
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
