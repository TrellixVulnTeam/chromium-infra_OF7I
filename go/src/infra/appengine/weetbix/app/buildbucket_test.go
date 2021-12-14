// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	bbv1 "go.chromium.org/luci/common/api/buildbucket/buildbucket/v1"
	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/testutil"

	. "github.com/smartystreets/goconvey/convey"

	// Needed to ensure task class is registered.
	_ "infra/appengine/weetbix/internal/services/resultingester"
)

func TestHandleBuild(t *testing.T) {
	t.Parallel()
	ctx := testutil.TestingContext()
	ctx, _ = tq.TestingContext(ctx, nil)

	Convey(`Test BuildbucketPubSubHandler`, t, func() {
		Convey(`non chromium build is ignored`, func() {
			buildExp := bbv1.LegacyApiCommonBuildMessage{
				Project:   "fake",
				Bucket:    "luci.fake.bucket",
				Id:        87654321,
				Status:    bbv1.StatusCompleted,
				CreatedTs: bbv1.FormatTimestamp(time.Now()),
			}
			r := &http.Request{Body: makeBBReq(buildExp)}
			err := bbPubSubHandlerImpl(ctx, r)
			So(err, ShouldBeNil)
		})

		Convey(`chromium build is processed`, func() {
			buildExp := bbv1.LegacyApiCommonBuildMessage{
				Project:   "chromium",
				Bucket:    chromiumCIBucket,
				Id:        87654321,
				Status:    bbv1.StatusCompleted,
				CreatedTs: bbv1.FormatTimestamp(time.Now()),
			}
			r := &http.Request{Body: makeBBReq(buildExp)}
			err := bbPubSubHandlerImpl(ctx, r)
			So(err, ShouldBeNil)
		})
	})
}

func makeBBReq(build bbv1.LegacyApiCommonBuildMessage) io.ReadCloser {
	bmsg := struct {
		Build    bbv1.LegacyApiCommonBuildMessage `json:"build"`
		Hostname string                           `json:"hostname"`
	}{build, "hostname"}
	bm, _ := json.Marshal(bmsg)
	return makeReq(bm)
}
