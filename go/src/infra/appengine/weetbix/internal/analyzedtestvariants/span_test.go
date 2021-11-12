// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package analyzedtestvariants

import (
	"testing"
	"time"

	"cloud.google.com/go/spanner"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/testutil"
	"infra/appengine/weetbix/internal/testutil/insert"
	pb "infra/appengine/weetbix/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAnalyzedTestVariantSpan(t *testing.T) {
	Convey(`TestAnalyzedTestVariantSpan`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		realm := "chromium:ci"
		status := pb.AnalyzedTestVariantStatus_FLAKY
		now := clock.Now(ctx).UTC()
		ps := []pb.AnalyzedTestVariantStatus{
			pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED,
			pb.AnalyzedTestVariantStatus_FLAKY,
		}
		puts := []time.Time{
			now.Add(-24 * time.Hour),
			now.Add(-240 * time.Hour),
		}
		builder := "builder"
		ms := []*spanner.Mutation{
			insert.AnalyzedTestVariant(realm, "ninja://test1", "variantHash1", status,
				map[string]interface{}{
					"Builder":                   builder,
					"StatusUpdateTime":          now.Add(-time.Hour),
					"PreviousStatuses":          ps,
					"PreviousStatusUpdateTimes": puts,
				}),
			insert.AnalyzedTestVariant(realm, "ninja://test1", "variantHash2", pb.AnalyzedTestVariantStatus_HAS_UNEXPECTED_RESULTS, map[string]interface{}{
				"Builder": builder,
			}),
			insert.AnalyzedTestVariant(realm, "ninja://test2", "variantHash1", status,
				map[string]interface{}{
					"Builder": "anotherbuilder",
				}),
			insert.AnalyzedTestVariant(realm, "ninja://test3", "variantHash", status, nil),
			insert.AnalyzedTestVariant(realm, "ninja://test4", "variantHash", pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED,
				map[string]interface{}{
					"Builder": builder,
				}),
			insert.AnalyzedTestVariant("anotherrealm", "ninja://test1", "variantHash1", status,
				map[string]interface{}{
					"Builder": builder,
				}),
		}
		testutil.MustApply(ctx, ms...)

		Convey(`TestReadStatus`, func() {
			ks := spanner.KeySets(
				spanner.Key{realm, "ninja://test1", "variantHash1"},
				spanner.Key{realm, "ninja://test1", "variantHash2"},
				spanner.Key{realm, "ninja://test-not-exists", "variantHash1"},
			)
			atvs := make([]*pb.AnalyzedTestVariant, 0)
			err := ReadStatus(span.Single(ctx), ks, func(atv *pb.AnalyzedTestVariant) error {
				So(atv.Realm, ShouldEqual, realm)
				atvs = append(atvs, atv)
				return nil
			})
			So(err, ShouldBeNil)
			So(len(atvs), ShouldEqual, 2)
		})

		Convey(`TestReadStatusHistory`, func() {
			exp := &StatusHistory{
				Status:                    status,
				StatusUpdateTime:          now.Add(-time.Hour),
				PreviousStatuses:          ps,
				PreviousStatusUpdateTimes: puts,
			}

			si, enqTime, err := ReadStatusHistory(span.Single(ctx), spanner.Key{realm, "ninja://test1", "variantHash1"})
			So(err, ShouldBeNil)
			So(si, ShouldResemble, exp)
			So(enqTime, ShouldResemble, spanner.NullTime{})
		})

		Convey(`TestQueryTestVariantsByBuilder`, func() {
			atvs := make([]*pb.AnalyzedTestVariant, 0)
			err := QueryTestVariantsByBuilder(span.Single(ctx), realm, builder, func(atv *pb.AnalyzedTestVariant) error {
				atvs = append(atvs, atv)
				return nil
			})
			So(err, ShouldBeNil)
			So(len(atvs), ShouldEqual, 2)
		})
	})
}
