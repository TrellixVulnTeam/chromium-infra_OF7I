// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
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
