// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/registration"
)

func TestCreateRackLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("CreateRackLSE", t, func() {
		Convey("Create new rackLSE with non existing racks", func() {
			rackLSE1 := &ufspb.RackLSE{
				Name:  "racklse-1",
				Racks: []string{"rack-1", "rack-2"},
			}
			resp, err := CreateRackLSE(ctx, rackLSE1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotCreate)
		})
		Convey("Create new rackLSE with existing racks", func() {
			rack1 := &ufspb.Rack{
				Name: "rack-1",
			}
			mresp, merr := registration.CreateRack(ctx, rack1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, rack1)

			rackLSE2 := &ufspb.RackLSE{
				Name:  "racklse-2",
				Racks: []string{"rack-1"},
			}
			resp, err := CreateRackLSE(ctx, rackLSE2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSE2)
		})
	})
}
