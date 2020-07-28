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

func mockMachineLSEPrototype(id string) *ufspb.MachineLSEPrototype {
	return &ufspb.MachineLSEPrototype{
		Name: id,
	}
}

func TestListMachineLSEPrototypes(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	machineLSEPrototypes := make([]*ufspb.MachineLSEPrototype, 0, 4)
	for i := 0; i < 4; i++ {
		machineLSEPrototype1 := mockMachineLSEPrototype("")
		machineLSEPrototype1.Name = fmt.Sprintf("machineLSEPrototype-%d", i)
		resp, _ := configuration.CreateMachineLSEPrototype(ctx, machineLSEPrototype1)
		machineLSEPrototypes = append(machineLSEPrototypes, resp)
	}
	Convey("ListMachineLSEPrototypes", t, func() {
		Convey("List MachineLSEPrototypes - filter invalid", func() {
			_, _, err := ListMachineLSEPrototypes(ctx, 5, "", "machine=mx-1", false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Failed to read filter for listing machinelseprototypes")
		})

		Convey("ListMachineLSEPrototypes - Full listing - happy path", func() {
			resp, _, _ := ListMachineLSEPrototypes(ctx, 5, "", "", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototypes)
		})
	})
}

func TestDeleteMachineLSEPrototype(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	machineLSEPrototype1 := mockMachineLSEPrototype("machineLSEPrototype-1")
	machineLSEPrototype2 := mockMachineLSEPrototype("machineLSEPrototype-2")
	Convey("DeleteMachineLSEPrototype", t, func() {
		Convey("Delete machineLSEPrototype by existing ID with machinelse reference", func() {
			resp, cerr := configuration.CreateMachineLSEPrototype(ctx, machineLSEPrototype1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype1)

			machineLSE1 := &ufspb.MachineLSE{
				Name:                "machinelse-1",
				MachineLsePrototype: "machineLSEPrototype-1",
			}
			mresp, merr := inventory.CreateMachineLSE(ctx, machineLSE1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, machineLSE1)

			err := DeleteMachineLSEPrototype(ctx, "machineLSEPrototype-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = configuration.GetMachineLSEPrototype(ctx, "machineLSEPrototype-1")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype1)
		})
		Convey("Delete machineLSEPrototype successfully by existing ID without references", func() {
			resp, cerr := configuration.CreateMachineLSEPrototype(ctx, machineLSEPrototype2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype2)

			err := DeleteMachineLSEPrototype(ctx, "machineLSEPrototype-2")
			So(err, ShouldBeNil)

			resp, cerr = configuration.GetMachineLSEPrototype(ctx, "machineLSEPrototype-2")
			So(resp, ShouldBeNil)
			So(cerr, ShouldNotBeNil)
			So(cerr.Error(), ShouldContainSubstring, NotFound)
		})
	})
}
