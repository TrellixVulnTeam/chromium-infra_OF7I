// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"context"
	"testing"
	"time"

	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/testutil"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestSpan(t *testing.T) {
	Convey(`With Spanner Test Database`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		Convey(`ReadActive`, func() {
			Convey(`Empty`, func() {
				SetRulesForTesting(ctx, nil)

				rules, err := ReadActive(span.Single(ctx), testProject)
				So(err, ShouldBeNil)
				So(rules, ShouldResemble, []*FailureAssociationRule{})
			})
			Convey(`Multiple`, func() {
				rulesToCreate := []*FailureAssociationRule{
					NewRule(0).Build(),
					NewRule(1).WithProject("otherproject").Build(),
					NewRule(2).WithActive(false).Build(),
					NewRule(3).Build(),
				}
				SetRulesForTesting(ctx, rulesToCreate)

				rules, err := ReadActive(span.Single(ctx), testProject)
				So(err, ShouldBeNil)
				So(rules, ShouldResemble, []*FailureAssociationRule{
					rulesToCreate[0],
					rulesToCreate[3],
				})
			})
		})
		Convey(`ReadDelta`, func() {
			Convey(`Invalid since time`, func() {
				_, err := ReadDelta(span.Single(ctx), testProject, time.Time{})
				So(err, ShouldErrLike, "cannot query rule deltas from before project inception")
			})
			Convey(`Empty`, func() {
				SetRulesForTesting(ctx, nil)
				rules, err := ReadDelta(span.Single(ctx), testProject, StartingEpoch)
				So(err, ShouldBeNil)
				So(rules, ShouldResemble, []*FailureAssociationRule{})
			})
			Convey(`Multiple`, func() {
				reference := time.Date(2020, 1, 2, 3, 4, 5, 6000, time.UTC)
				rulesToCreate := []*FailureAssociationRule{
					NewRule(0).WithLastUpdated(reference).Build(),
					NewRule(1).WithProject("otherproject").WithLastUpdated(reference.Add(time.Minute)).Build(),
					NewRule(2).WithActive(false).WithLastUpdated(reference.Add(time.Minute)).Build(),
					NewRule(3).WithLastUpdated(reference.Add(time.Microsecond)).Build(),
				}
				SetRulesForTesting(ctx, rulesToCreate)

				rules, err := ReadDelta(span.Single(ctx), testProject, StartingEpoch)
				So(err, ShouldBeNil)
				So(rules, ShouldResemble, []*FailureAssociationRule{
					rulesToCreate[0],
					rulesToCreate[2],
					rulesToCreate[3],
				})

				rules, err = ReadDelta(span.Single(ctx), testProject, reference)
				So(err, ShouldBeNil)
				So(rules, ShouldResemble, []*FailureAssociationRule{
					rulesToCreate[2],
					rulesToCreate[3],
				})

				rules, err = ReadDelta(span.Single(ctx), testProject, reference.Add(time.Minute))
				So(err, ShouldBeNil)
				So(rules, ShouldResemble, []*FailureAssociationRule{})
			})
		})
		Convey(`ReadLastUpdated`, func() {
			Convey(`Empty`, func() {
				SetRulesForTesting(ctx, nil)

				timestamp, err := ReadLastUpdated(span.Single(ctx), testProject)
				So(err, ShouldBeNil)
				So(timestamp, ShouldEqual, StartingEpoch)
			})
			Convey(`Multiple`, func() {
				// Spanner commit timestamps are in microsecond
				// (not nanosecond) granularity. The MAX operator
				// on timestamps truncates to microseconds. For this
				// reason, we use microsecond resolution timestamps
				// when testing.
				reference := time.Date(2020, 1, 2, 3, 4, 5, 6000, time.UTC)
				rulesToCreate := []*FailureAssociationRule{
					NewRule(0).WithLastUpdated(reference.Add(-1 * time.Hour)).Build(),
					NewRule(1).WithProject("otherproject").WithLastUpdated(reference.Add(time.Hour)).Build(),
					NewRule(2).WithActive(false).WithLastUpdated(reference).Build(),
					NewRule(3).WithLastUpdated(reference.Add(-2 * time.Hour)).Build(),
				}
				SetRulesForTesting(ctx, rulesToCreate)

				timestamp, err := ReadLastUpdated(span.Single(ctx), testProject)
				So(err, ShouldBeNil)
				So(timestamp, ShouldEqual, reference)
			})
		})
		Convey(`Create`, func() {
			testCreate := func(bc *FailureAssociationRule) error {
				_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
					return Create(ctx, bc)
				})
				return err
			}
			r := NewRule(100).Build()
			Convey(`Valid`, func() {
				testExists := func(expectedRule *FailureAssociationRule) {
					txn, cancel := span.ReadOnlyTransaction(ctx)
					defer cancel()
					rules, err := ReadActive(txn, testProject)

					So(err, ShouldBeNil)
					So(len(rules), ShouldEqual, 1)

					readRule := rules[0]
					So(readRule.CreationTime, ShouldNotBeZeroValue)
					So(readRule.LastUpdated, ShouldNotBeZeroValue)
					So(readRule.LastUpdated, ShouldEqual, readRule.CreationTime)
					// CreationTime and LastUpdated are set by implementation.
					// Ignore their values when comparing to expected rules.
					readRule.LastUpdated = expectedRule.LastUpdated
					readRule.CreationTime = expectedRule.CreationTime
					So(readRule, ShouldResemble, expectedRule)
				}

				Convey(`With Source Cluster`, func() {
					So(r.SourceCluster.Algorithm, ShouldNotBeEmpty)
					So(r.SourceCluster.ID, ShouldNotBeNil)
					err := testCreate(r)
					So(err, ShouldBeNil)

					testExists(r)
				})
				Convey(`Without Source Cluster`, func() {
					// E.g. in case of a manually created rule.
					r.SourceCluster = clustering.ClusterID{}
					err := testCreate(r)
					So(err, ShouldBeNil)

					testExists(r)
				})
			})
			Convey(`With invalid Project`, func() {
				Convey(`Missing`, func() {
					r.Project = ""
					err := testCreate(r)
					So(err, ShouldErrLike, "project must be valid")
				})
				Convey(`Invalid`, func() {
					r.Project = "!"
					err := testCreate(r)
					So(err, ShouldErrLike, "project must be valid")
				})
			})
			Convey(`With invalid Rule Definition`, func() {
				r.RuleDefinition = "invalid"
				err := testCreate(r)
				So(err, ShouldErrLike, "rule definition is not valid")
			})
			Convey(`With invalid Bug`, func() {
				r.Bug.System = ""
				err := testCreate(r)
				So(err, ShouldErrLike, "bug is not valid")
			})
			Convey(`With invalid Source Cluster`, func() {
				So(r.SourceCluster.ID, ShouldNotBeNil)
				r.SourceCluster.Algorithm = ""
				err := testCreate(r)
				So(err, ShouldErrLike, "source cluster ID is not valid")
			})
		})
	})
}
