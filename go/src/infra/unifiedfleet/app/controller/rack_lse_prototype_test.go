// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	fleet "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
)

func mockRackLSEPrototype(id string) *fleet.RackLSEPrototype {
	return &fleet.RackLSEPrototype{
		Name: id,
	}
}

func TestDeleteRackLSEPrototype(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	rackLSEPrototype1 := mockRackLSEPrototype("rackLSEPrototype-1")
	rackLSEPrototype2 := mockRackLSEPrototype("rackLSEPrototype-2")
	Convey("DeleteRackLSEPrototype", t, func() {
		Convey("Delete rackLSEPrototype by existing ID with racklse reference", func() {
			resp, cerr := configuration.CreateRackLSEPrototype(ctx, rackLSEPrototype1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype1)

			rackLSE1 := &fleet.RackLSE{
				Name:             "racklse-1",
				RackLsePrototype: "rackLSEPrototype-1",
			}
			mresp, merr := inventory.CreateRackLSE(ctx, rackLSE1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, rackLSE1)

			err := DeleteRackLSEPrototype(ctx, "rackLSEPrototype-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = configuration.GetRackLSEPrototype(ctx, "rackLSEPrototype-1")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype1)
		})
		Convey("Delete rackLSEPrototype successfully by existing ID without references", func() {
			resp, cerr := configuration.CreateRackLSEPrototype(ctx, rackLSEPrototype2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype2)

			err := DeleteRackLSEPrototype(ctx, "rackLSEPrototype-2")
			So(err, ShouldBeNil)

			resp, cerr = configuration.GetRackLSEPrototype(ctx, "rackLSEPrototype-2")
			So(resp, ShouldBeNil)
			So(cerr, ShouldNotBeNil)
			So(cerr.Error(), ShouldContainSubstring, NotFound)
		})
	})
}
