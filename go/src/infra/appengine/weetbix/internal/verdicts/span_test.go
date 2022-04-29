// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package verdicts

import (
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/protobuf/types/known/durationpb"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/testutil"
	"infra/appengine/weetbix/internal/testutil/insert"
	pb "infra/appengine/weetbix/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestComputeTestVariantStatusFromVerdicts(t *testing.T) {
	Convey(`ComputeTestVariantStatusFromVerdicts`, t, func() {
		ctx := testutil.SpannerTestContext(t)

		realm := "chromium:ci"
		status := pb.AnalyzedTestVariantStatus_FLAKY
		vh := "varianthash"

		// Insert parent AnalyzedTestVariants.
		ms := []*spanner.Mutation{
			insert.AnalyzedTestVariant(realm, "ninja://still_flaky", vh, status, nil),
			insert.AnalyzedTestVariant(realm, "ninja://consistently_expected", vh, status, nil),
			insert.AnalyzedTestVariant(realm, "ninja://consistently_unexpected", vh, status, nil),
			insert.AnalyzedTestVariant(realm, "ninja://has_unexpected_results", vh, status, nil),
			insert.AnalyzedTestVariant(realm, "ninja://no_new_results", vh, status, nil),
		}
		testutil.MustApply(ctx, ms...)

		test := func(tID string, expStatus pb.AnalyzedTestVariantStatus) {
			ctx, cancel := span.ReadOnlyTransaction(ctx)
			defer cancel()

			tvKey := &taskspb.TestVariantKey{
				Realm:       realm,
				TestId:      tID,
				VariantHash: vh,
			}
			tvStatus, err := ComputeTestVariantStatusFromVerdicts(ctx, tvKey, durationpb.New(24*time.Hour))
			So(err, ShouldBeNil)
			So(tvStatus, ShouldEqual, expStatus)
		}

		Convey(`still_flaky`, func() {
			tID := "ninja://still_flaky"
			ms := []*spanner.Mutation{
				insert.Verdict(realm, tID, vh, "build-0", internal.VerdictStatus_EXPECTED, clock.Now(ctx).UTC().Add(-time.Hour), nil),
				insert.Verdict(realm, tID, vh, "build-1", internal.VerdictStatus_VERDICT_FLAKY, clock.Now(ctx).UTC().Add(-2*time.Hour), nil),
				insert.Verdict(realm, tID, vh, "build-2", internal.VerdictStatus_VERDICT_FLAKY, clock.Now(ctx).UTC().Add(-3*time.Hour), nil),
				insert.Verdict(realm, tID, vh, "build-3", internal.VerdictStatus_UNEXPECTED, clock.Now(ctx).UTC().Add(-4*time.Hour), nil),
			}
			testutil.MustApply(ctx, ms...)

			test(tID, pb.AnalyzedTestVariantStatus_FLAKY)
		})

		Convey(`no_new_results`, func() {
			tID := "ninja://no_new_results"
			ms := []*spanner.Mutation{
				insert.Verdict(realm, tID, vh, "build-0", internal.VerdictStatus_EXPECTED, clock.Now(ctx).UTC().Add(-25*time.Hour), nil),
			}
			testutil.MustApply(ctx, ms...)
			test(tID, pb.AnalyzedTestVariantStatus_NO_NEW_RESULTS)
		})

		Convey(`consistently_unexpected`, func() {
			tID := "ninja://consistently_unexpected"
			ms := []*spanner.Mutation{
				insert.Verdict(realm, tID, vh, "build-0", internal.VerdictStatus_VERDICT_FLAKY, clock.Now(ctx).UTC().Add(-26*time.Hour), nil),
				insert.Verdict(realm, tID, vh, "build-1", internal.VerdictStatus_UNEXPECTED, clock.Now(ctx).UTC().Add(-time.Hour), nil),
				insert.Verdict(realm, tID, vh, "build-2", internal.VerdictStatus_UNEXPECTED, clock.Now(ctx).UTC().Add(-2*time.Hour), nil),
			}
			testutil.MustApply(ctx, ms...)
			test(tID, pb.AnalyzedTestVariantStatus_CONSISTENTLY_UNEXPECTED)
		})

		Convey(`consistently_expected`, func() {
			tID := "ninja://consistently_expected"
			ms := []*spanner.Mutation{
				insert.Verdict(realm, tID, vh, "build-0", internal.VerdictStatus_EXPECTED, clock.Now(ctx).UTC().Add(-time.Hour), nil),
				insert.Verdict(realm, tID, vh, "build-1", internal.VerdictStatus_EXPECTED, clock.Now(ctx).UTC().Add(-2*time.Hour), nil),
			}
			testutil.MustApply(ctx, ms...)
			test(tID, pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED)
		})

		Convey(`has_unexpected_results`, func() {
			tID := "ninja://has_unexpected_results"
			ms := []*spanner.Mutation{
				insert.Verdict(realm, tID, vh, "build-0", internal.VerdictStatus_EXPECTED, clock.Now(ctx).UTC().Add(-time.Hour), nil),
				insert.Verdict(realm, tID, vh, "build-1", internal.VerdictStatus_UNEXPECTED, clock.Now(ctx).UTC().Add(-2*time.Hour), nil),
			}
			testutil.MustApply(ctx, ms...)
			test(tID, pb.AnalyzedTestVariantStatus_HAS_UNEXPECTED_RESULTS)
		})
	})
}
