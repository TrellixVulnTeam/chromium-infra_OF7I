// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package args contains the logic for assembling all data required for
// creating an individual task request.
package args

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
)

func TestDisplayNameTagsForUnamedRequest(t *testing.T) {
	Convey("Given a request does not specify a display name", t, func() {
		ctx := context.Background()
		inv := basicInvocation()
		setTestName(inv, "foo-name")
		var params test_platform.Request_Params
		var dummyWorkerConfig = &config.Config_SkylabWorker{}
		setBuild(&params, "foo-build")
		setRequestKeyval(&params, "suite", "foo-suite")
		setRequestMaximumDuration(&params, 1000)
		Convey("when generating a test runner request's args", func() {
			g := NewGenerator(inv, &params, dummyWorkerConfig, "", noDeadline)
			got, err := g.GenerateArgs(ctx)
			So(err, ShouldBeNil)
			Convey("the display name tag is generated correctly.", func() {
				So(got.SwarmingTags, ShouldContain, "display_name:foo-build/foo-suite/foo-name")
			})
		})
	})
}
