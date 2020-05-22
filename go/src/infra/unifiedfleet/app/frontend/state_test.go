// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/genproto/googleapis/rpc/code"

	api "infra/unifiedfleet/api/v1/rpc"
)

func TestImportStates(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Import machine lses", t, func() {
		Convey("happy path", func() {
			req := &api.ImportStatesRequest{
				Source: &api.ImportStatesRequest_MachineDbSource{
					MachineDbSource: &api.MachineDBSource{
						Host: "fake_host",
					},
				},
			}
			res, err := tf.Fleet.ImportStates(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)
		})
	})
}
