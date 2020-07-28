// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
)

func mockRackLSEPrototype(id string) *ufspb.RackLSEPrototype {
	return &ufspb.RackLSEPrototype{
		Name: id,
	}
}

func TestListRackLSEPrototypes(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	rackLSEPrototypes := make([]*ufspb.RackLSEPrototype, 0, 4)
	for i := 0; i < 4; i++ {
		rackLSEPrototype1 := mockRackLSEPrototype("")
		rackLSEPrototype1.Name = fmt.Sprintf("rackLSEPrototype-%d", i)
		resp, _ := configuration.CreateRackLSEPrototype(ctx, rackLSEPrototype1)
		rackLSEPrototypes = append(rackLSEPrototypes, resp)
	}
	Convey("ListRackLSEPrototypes", t, func() {
		Convey("List RackLSEPrototypes - filter invalid", func() {
			_, _, err := ListRackLSEPrototypes(ctx, 5, "", "machine=mx-1", false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Failed to read filter for listing racklseprototypes")
		})

		Convey("ListRackLSEPrototypes - Full listing - happy path", func() {
			resp, _, _ := ListRackLSEPrototypes(ctx, 5, "", "", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototypes)
		})
	})
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

			rackLSE1 := &ufspb.RackLSE{
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
