// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	proto "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/registration"
)

func TestCreateMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("CreateMachineLSE", t, func() {
		Convey("Create new machineLSE with non existing machines", func() {
			machineLSE1 := &proto.MachineLSE{
				Name:     "machinelse-1",
				Machines: []string{"machine-1", "machine-2"},
			}
			resp, err := CreateMachineLSE(ctx, machineLSE1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotCreate)
		})
		Convey("Create new machineLSE with existing machines", func() {
			machine1 := &proto.Machine{
				Name: "machine-1",
			}
			mresp, merr := registration.CreateMachine(ctx, machine1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, machine1)

			machineLSE2 := &proto.MachineLSE{
				Name:     "machinelse-2",
				Machines: []string{"machine-1"},
			}
			resp, err := CreateMachineLSE(ctx, machineLSE2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE2)
		})
	})
}
