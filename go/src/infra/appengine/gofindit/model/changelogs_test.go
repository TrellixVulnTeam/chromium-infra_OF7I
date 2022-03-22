// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package model

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestChangeLogs(t *testing.T) {
	Convey("GetReviewUrl", t, func() {
		cl := &ChangeLog{
			Message: "",
		}
		_, err := cl.GetReviewUrl()
		So(err, ShouldNotBeNil)
		cl = &ChangeLog{
			Message: "Use TestActivationManager for all page activations\n\nblah blah\n\nChange-Id: blah\nBug: blah\nReviewed-on: https://chromium-review.googlesource.com/c/chromium/src/+/3472129\nReviewed-by: blah blah\n",
		}
		reviewUrl, err := cl.GetReviewUrl()
		So(err, ShouldBeNil)
		So(reviewUrl, ShouldEqual, "https://chromium-review.googlesource.com/c/chromium/src/+/3472129")
	})
}
