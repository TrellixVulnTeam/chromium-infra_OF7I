// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package analyzedtestvariants

import (
	"testing"

	"cloud.google.com/go/spanner"
	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/testutil"
	"infra/appengine/weetbix/internal/testutil/insert"
	pb "infra/appengine/weetbix/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRead(t *testing.T) {
	Convey(`TestRead`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		realm := "chromium:ci"
		status := pb.AnalyzedTestVariantStatus_FLAKY
		ms := []*spanner.Mutation{
			insert.AnalyzedTestVariant(realm, "ninja://test1", "variantHash1", status, nil),
			insert.AnalyzedTestVariant(realm, "ninja://test1", "variantHash2", status, nil),
			insert.AnalyzedTestVariant(realm, "ninja://test2", "variantHash1", status, nil),
			insert.AnalyzedTestVariant(realm, "ninja://test3", "variantHash", status, nil),
		}
		testutil.MustApply(ctx, ms...)
		ctx, cancel := span.ReadOnlyTransaction(ctx)
		defer cancel()

		ks := spanner.KeySets(
			spanner.Key{realm, "ninja://test1", "variantHash1"},
			spanner.Key{realm, "ninja://test1", "variantHash2"},
			spanner.Key{realm, "ninja://test-not-exists", "variantHash1"},
		)
		atvs := make([]*pb.AnalyzedTestVariant, 0)
		err := Read(ctx, ks, func(atv *pb.AnalyzedTestVariant) error {
			So(atv.Realm, ShouldEqual, realm)
			atvs = append(atvs, atv)
			return nil
		})
		So(err, ShouldBeNil)
		So(len(atvs), ShouldEqual, 2)
	})
}
