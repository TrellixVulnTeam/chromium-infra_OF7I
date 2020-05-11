// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"fmt"
	"testing"

	proto "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/util"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func mockMachineLSE(id string) *proto.MachineLSE {
	return &proto.MachineLSE{
		Name: util.AddPrefix(machineLSECollection, id),
	}
}

func TestCreateMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	machineLSE1 := mockMachineLSE("machineLSE-1")
	machineLSE2 := mockMachineLSE("machineLSE-2")
	Convey("CreateMachineLSEs", t, func() {
		Convey("Create new machineLSE with machineLSE_id", func() {
			req := &api.CreateMachineLSERequest{
				MachineLSE:   machineLSE1,
				MachineLSEId: "machineLSE-1",
			}
			resp, err := tf.Fleet.CreateMachineLSE(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE1)
		})

		Convey("Create existing machineLSEs", func() {
			req := &api.CreateMachineLSERequest{
				MachineLSE:   machineLSE1,
				MachineLSEId: "machineLSE-1",
			}
			resp, err := tf.Fleet.CreateMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})

		Convey("Create new machineLSE - Invalid input nil", func() {
			req := &api.CreateMachineLSERequest{
				MachineLSE: nil,
			}
			resp, err := tf.Fleet.CreateMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Create new machineLSE - Invalid input empty ID", func() {
			req := &api.CreateMachineLSERequest{
				MachineLSE:   machineLSE2,
				MachineLSEId: "",
			}
			resp, err := tf.Fleet.CreateMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyID)
		})

		Convey("Create new machineLSE - Invalid input invalid characters", func() {
			req := &api.CreateMachineLSERequest{
				MachineLSE:   machineLSE2,
				MachineLSEId: "a.b)7&",
			}
			resp, err := tf.Fleet.CreateMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestUpdateMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	machineLSE1 := mockMachineLSE("machineLSE-1")
	machineLSE2 := mockMachineLSE("machineLSE-1")
	machineLSE2.Hostname = "Linux Server"
	machineLSE3 := mockMachineLSE("machineLSE-3")
	machineLSE4 := mockMachineLSE("")
	machineLSE5 := mockMachineLSE("a.b)7&")
	Convey("UpdateMachineLSEs", t, func() {
		Convey("Update existing machineLSEs", func() {
			req := &api.CreateMachineLSERequest{
				MachineLSE:   machineLSE1,
				MachineLSEId: "machineLSE-1",
			}
			resp, err := tf.Fleet.CreateMachineLSE(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE1)
			ureq := &api.UpdateMachineLSERequest{
				MachineLSE: machineLSE2,
			}
			resp, err = tf.Fleet.UpdateMachineLSE(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE2)
		})

		Convey("Update non-existing machineLSEs", func() {
			ureq := &api.UpdateMachineLSERequest{
				MachineLSE: machineLSE3,
			}
			resp, err := tf.Fleet.UpdateMachineLSE(tf.C, ureq)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Update machineLSE - Invalid input nil", func() {
			req := &api.UpdateMachineLSERequest{
				MachineLSE: nil,
			}
			resp, err := tf.Fleet.UpdateMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Update machineLSE - Invalid input empty name", func() {
			machineLSE4.Name = ""
			req := &api.UpdateMachineLSERequest{
				MachineLSE: machineLSE4,
			}
			resp, err := tf.Fleet.UpdateMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Update machineLSE - Invalid input invalid characters", func() {
			req := &api.UpdateMachineLSERequest{
				MachineLSE: machineLSE5,
			}
			resp, err := tf.Fleet.UpdateMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestGetMachineLSE(t *testing.T) {
	t.Parallel()
	Convey("GetMachineLSE", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		machineLSE1 := mockMachineLSE("machineLSE-1")
		req := &api.CreateMachineLSERequest{
			MachineLSE:   machineLSE1,
			MachineLSEId: "machineLSE-1",
		}
		resp, err := tf.Fleet.CreateMachineLSE(tf.C, req)
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, machineLSE1)
		Convey("Get machineLSE by existing ID", func() {
			req := &api.GetMachineLSERequest{
				Name: util.AddPrefix(machineLSECollection, "machineLSE-1"),
			}
			resp, err := tf.Fleet.GetMachineLSE(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE1)
		})
		Convey("Get machineLSE by non-existing ID", func() {
			req := &api.GetMachineLSERequest{
				Name: util.AddPrefix(machineLSECollection, "machineLSE-2"),
			}
			resp, err := tf.Fleet.GetMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get machineLSE - Invalid input empty name", func() {
			req := &api.GetMachineLSERequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})
		Convey("Get machineLSE - Invalid input invalid characters", func() {
			req := &api.GetMachineLSERequest{
				Name: util.AddPrefix(machineLSECollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestListMachineLSEs(t *testing.T) {
	t.Parallel()
	Convey("ListMachineLSEs", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		machineLSEs := make([]*proto.MachineLSE, 0, 4)
		for i := 0; i < 4; i++ {
			machineLSE1 := mockMachineLSE("machineLSE-1")
			req := &api.CreateMachineLSERequest{
				MachineLSE:   machineLSE1,
				MachineLSEId: fmt.Sprintf("machineLSE-%d", i),
			}
			resp, err := tf.Fleet.CreateMachineLSE(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE1)
			machineLSEs = append(machineLSEs, resp)
		}

		Convey("ListMachineLSEs - page_size negative", func() {
			req := &api.ListMachineLSEsRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListMachineLSEs(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidPageSize)
		})

		Convey("ListMachineLSEs - page_token invalid", func() {
			req := &api.ListMachineLSEsRequest{
				PageSize:  5,
				PageToken: "abc",
			}
			resp, err := tf.Fleet.ListMachineLSEs(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("ListMachineLSEs - Full listing Max PageSize", func() {
			req := &api.ListMachineLSEsRequest{
				PageSize: 2000,
			}
			resp, err := tf.Fleet.ListMachineLSEs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.MachineLSEs, ShouldResembleProto, machineLSEs)
		})

		Convey("ListMachineLSEs - Full listing with no pagination", func() {
			req := &api.ListMachineLSEsRequest{
				PageSize: 0,
			}
			resp, err := tf.Fleet.ListMachineLSEs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.MachineLSEs, ShouldResembleProto, machineLSEs)
		})

		Convey("ListMachineLSEs - listing with pagination", func() {
			req := &api.ListMachineLSEsRequest{
				PageSize: 3,
			}
			resp, err := tf.Fleet.ListMachineLSEs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.MachineLSEs, ShouldResembleProto, machineLSEs[:3])

			req = &api.ListMachineLSEsRequest{
				PageSize:  3,
				PageToken: resp.NextPageToken,
			}
			resp, err = tf.Fleet.ListMachineLSEs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.MachineLSEs, ShouldResembleProto, machineLSEs[3:])
		})
	})
}

func TestDeleteMachineLSE(t *testing.T) {
	t.Parallel()
	Convey("DeleteMachineLSE", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		machineLSE1 := mockMachineLSE("")
		req := &api.CreateMachineLSERequest{
			MachineLSE:   machineLSE1,
			MachineLSEId: "machineLSE-1",
		}
		resp, err := tf.Fleet.CreateMachineLSE(tf.C, req)
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, machineLSE1)
		Convey("Delete machineLSE by existing ID", func() {
			req := &api.DeleteMachineLSERequest{
				Name: util.AddPrefix(machineLSECollection, "machineLSE-1"),
			}
			_, err := tf.Fleet.DeleteMachineLSE(tf.C, req)
			So(err, ShouldBeNil)
			greq := &api.GetMachineLSERequest{
				Name: util.AddPrefix(machineLSECollection, "machineLSE-1"),
			}
			res, err := tf.Fleet.GetMachineLSE(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete machineLSE by non-existing ID", func() {
			req := &api.DeleteMachineLSERequest{
				Name: util.AddPrefix(machineLSECollection, "machineLSE-2"),
			}
			_, err := tf.Fleet.DeleteMachineLSE(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete machineLSE - Invalid input empty name", func() {
			req := &api.DeleteMachineLSERequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})
		Convey("Delete machineLSE - Invalid input invalid characters", func() {
			req := &api.DeleteMachineLSERequest{
				Name: util.AddPrefix(machineLSECollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}
