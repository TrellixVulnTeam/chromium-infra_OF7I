// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package state

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/proto"
)

func mockState(resource string, state ufspb.State) *ufspb.StateRecord {
	return &ufspb.StateRecord{
		ResourceName: resource,
		State:        state,
	}
}

func TestImportStateRecords(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	Convey("import states", t, func() {
		states := []*ufspb.StateRecord{
			mockState("machines/abc", ufspb.State_STATE_SERVING),
			mockState("vms/abc-1", ufspb.State_STATE_SERVING),
			mockState("vms/abc-2", ufspb.State_STATE_NEEDS_REPAIR),
		}
		Convey("happy path", func() {
			resp, err := ImportStateRecords(ctx, states)
			So(err, ShouldBeNil)
			So(resp.Passed(), ShouldHaveLength, len(states))
			getRes, _, err := ListStateRecords(ctx, 100, "")
			So(err, ShouldBeNil)
			So(getRes, ShouldResembleProto, states)
		})
	})
}
