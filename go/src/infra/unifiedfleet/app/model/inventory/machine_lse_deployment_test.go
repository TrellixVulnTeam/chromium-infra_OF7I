// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/models"
)

func mockMachineLSEDeployment(id string) *ufspb.MachineLSEDeployment {
	return &ufspb.MachineLSEDeployment{
		SerialNumber: id,
	}
}

func TestUpdateMachineLSEDeployment(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	Convey("UpdateMachineLSE", t, func() {
		Convey("Update non-existing machineLSEDeployment", func() {
			md1 := mockMachineLSEDeployment("serial-1")
			resp, err := UpdateMachineLSEDeployments(ctx, []*ufspb.MachineLSEDeployment{md1})
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 1)
			So(resp[0], ShouldResembleProto, md1)
		})

		Convey("Update existing machineLSEDeployment", func() {
			md2 := mockMachineLSEDeployment("serial-2")
			resp, err := UpdateMachineLSEDeployments(ctx, []*ufspb.MachineLSEDeployment{md2})
			So(err, ShouldBeNil)

			md2.Hostname = "hostname-2"
			resp, err = UpdateMachineLSEDeployments(ctx, []*ufspb.MachineLSEDeployment{md2})
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 1)
			So(resp[0], ShouldResembleProto, md2)
		})

		Convey("Update machineLSEDeployment - invalid hostname", func() {
			md3 := mockMachineLSEDeployment("")
			resp, err := UpdateMachineLSEDeployments(ctx, []*ufspb.MachineLSEDeployment{md3})
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Empty")
		})
	})
}
