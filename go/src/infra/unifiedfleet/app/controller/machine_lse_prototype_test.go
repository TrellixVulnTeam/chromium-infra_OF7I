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

func TestBatchGetMachineLSEPrototypes(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("BatchGetMachineLSEPrototypes", t, func() {
		Convey("Batch get machine lse prototypes - happy path", func() {
			entities := make([]*ufspb.MachineLSEPrototype, 4)
			for i := 0; i < 4; i++ {
				entities[i] = &ufspb.MachineLSEPrototype{
					Name: fmt.Sprintf("machinelseprototype-batchGet-%d", i),
				}
			}
			_, err := configuration.BatchUpdateMachineLSEPrototypes(ctx, entities)
			So(err, ShouldBeNil)
			resp, err := configuration.BatchGetMachineLSEPrototypes(ctx, []string{"machinelseprototype-batchGet-0", "machinelseprototype-batchGet-1", "machinelseprototype-batchGet-2", "machinelseprototype-batchGet-3"})
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 4)
			So(resp, ShouldResembleProto, entities)
		})
		Convey("Batch get machine lse prototypes  - missing id", func() {
			resp, err := configuration.BatchGetMachineLSEPrototypes(ctx, []string{"machinelseprototype-batchGet-non-existing"})
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "machinelseprototype-batchGet-non-existing")
		})
		Convey("Batch get machine lse prototypes  - empty input", func() {
			resp, err := configuration.BatchGetMachineLSEPrototypes(ctx, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)

			input := make([]string, 0)
			resp, err = configuration.BatchGetMachineLSEPrototypes(ctx, input)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)
		})
	})
}
