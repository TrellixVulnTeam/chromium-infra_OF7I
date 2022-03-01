// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gob

import (
	"context"
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeClient struct {
	err   error
	max   int
	count int
}

func (c *fakeClient) op() error {
	c.count += 1
	if c.count <= c.max {
		return c.err
	}
	return nil
}

func TestRetry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = CtxForTest(ctx)

	Convey("Retry", t, func() {

		Convey("retries not found errors", func() {
			client := &fakeClient{
				err: status.Error(codes.NotFound, "fake not found failure"),
				max: 1,
			}

			err := Retry(ctx, "fake op", client.op)

			So(err, ShouldBeNil)
		})

		Convey("retries service not available errors", func() {
			client := &fakeClient{
				err: status.Error(codes.Unavailable, "fake unavailable failure"),
				max: 1,
			}

			err := Retry(ctx, "fake op", client.op)

			So(err, ShouldBeNil)
		})

		Convey("fails if all retries are exhausted", func() {
			client := &fakeClient{
				err: status.Error(codes.NotFound, "fake not found failure"),
				max: 6,
			}

			err := Retry(ctx, "fake op", client.op)

			So(err, ShouldErrLike, "fake not found failure")
		})

		Convey("does not retry other errors", func() {
			client := &fakeClient{
				err: errors.New("non-retriable error"),
				max: 1,
			}

			err := Retry(ctx, "fake op", client.op)

			So(err, ShouldErrLike, "non-retriable error")
			So(client.count, ShouldEqual, 1)
		})

	})
}
