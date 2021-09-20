// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
)

func mockServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		LastCheckedVMMacAddress: "000000",
	}
}

func TestGetLastCheckedVMMacAddress(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	sc := mockServiceConfig()
	Convey("GetLastCheckedVMMacAddress", t, func() {
		err := UpdateServiceConfig(ctx, sc)
		So(err, ShouldBeNil)
		resp, err := GetServiceConfig(ctx)
		So(err, ShouldBeNil)
		So(resp.LastCheckedVMMacAddress, ShouldEqual, "000000")
	})
}
