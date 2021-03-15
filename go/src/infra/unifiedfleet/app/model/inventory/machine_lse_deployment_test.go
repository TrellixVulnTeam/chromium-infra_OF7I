// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/models"
	. "infra/unifiedfleet/app/model/datastore"
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

func TestGetMachineLSEDeployment(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	dr1 := mockMachineLSEDeployment("dr-get-1")
	Convey("GetMachineLSEDeployment", t, func() {
		Convey("Get machine deployment record by existing ID", func() {
			resp, err := UpdateMachineLSEDeployments(ctx, []*ufspb.MachineLSEDeployment{dr1})
			So(err, ShouldBeNil)
			So(resp[0], ShouldResembleProto, dr1)
			respDr, err := GetMachineLSEDeployment(ctx, "dr-get-1")
			So(err, ShouldBeNil)
			So(respDr, ShouldResembleProto, dr1)
		})

		Convey("Get machine deployment record by non-existing ID", func() {
			resp, err := GetMachineLSEDeployment(ctx, "dr-get-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get machine deployment record - invalid ID", func() {
			resp, err := GetMachineLSEDeployment(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestBatchGetMachineLSEDeployments(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	Convey("BatchGetMachineLSEDeployments", t, func() {
		Convey("Batch get machine lse deployments - happy path", func() {
			drs := make([]*ufspb.MachineLSEDeployment, 4)
			for i := 0; i < 4; i++ {
				drs[i] = mockMachineLSEDeployment(fmt.Sprintf("dr-batchGet-%d", i))
			}
			_, err := UpdateMachineLSEDeployments(ctx, drs)
			So(err, ShouldBeNil)
			resp, err := BatchGetMachineLSEDeployments(ctx, []string{"dr-batchGet-0", "dr-batchGet-1", "dr-batchGet-2", "dr-batchGet-3"})
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 4)
			So(resp, ShouldResembleProto, drs)
		})

		Convey("Batch get machine lse deployments - missing id", func() {
			resp, err := BatchGetMachineLSEDeployments(ctx, []string{"dr-batchGet-non-existing"})
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "dr-batchGet-non-existing")
		})

		Convey("Batch get machine lse deployments - empty input", func() {
			resp, err := BatchGetMachineLSEDeployments(ctx, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)

			input := make([]string, 0)
			resp, err = BatchGetMachineLSEDeployments(ctx, input)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)
		})
	})
}
