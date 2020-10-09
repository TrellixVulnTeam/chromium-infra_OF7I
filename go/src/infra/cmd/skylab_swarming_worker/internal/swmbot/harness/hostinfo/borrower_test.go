// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package hostinfo

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"infra/cmd/skylab_swarming_worker/internal/autotest/hostinfo"
	"infra/cmd/skylab_swarming_worker/internal/swmbot"
)

func TestCopyToProvisioningLabels(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	Convey("Copy values when close", t, func() {
		Convey("Do not copy if nothing on both sides", func() {
			bi := &swmbot.LocalState{
				ProvisionableLabels:     make(map[string]string),
				ProvisionableAttributes: make(map[string]string),
			}
			b := &Borrower{
				hostInfo: &hostinfo.HostInfo{
					Labels:     []string{},
					Attributes: make(map[string]string),
				},
				botInfo: bi,
			}
			So(bi.ProvisionableLabels, ShouldHaveLength, 0)
			b.Close(ctx)
			So(bi.ProvisionableLabels, ShouldHaveLength, 0)
		})
		Convey("Set provisioning label when host-info provide info", func() {
			bi := &swmbot.LocalState{
				ProvisionableLabels:     make(map[string]string),
				ProvisionableAttributes: make(map[string]string),
			}
			b := &Borrower{
				hostInfo: &hostinfo.HostInfo{
					Labels:     []string{"cros-version:hi"},
					Attributes: make(map[string]string),
				},
				botInfo: bi,
			}
			So(bi.ProvisionableLabels, ShouldHaveLength, 0)
			b.Close(ctx)
			So(bi.ProvisionableLabels, ShouldHaveLength, 1)
			So(bi.ProvisionableLabels["cros-version"], ShouldEqual, "hi")
		})
		Convey("Update provisioning label when host-info provide info", func() {
			bi := &swmbot.LocalState{
				ProvisionableLabels: map[string]string{
					"cros-something": "hello",
					"cros-version":   "hello",
				},
				ProvisionableAttributes: make(map[string]string),
			}
			b := &Borrower{
				hostInfo: &hostinfo.HostInfo{
					Labels:     []string{"cros-version:hi"},
					Attributes: make(map[string]string),
				},
				botInfo: bi,
			}
			So(bi.ProvisionableLabels, ShouldHaveLength, 2)
			So(bi.ProvisionableLabels["cros-version"], ShouldEqual, "hello")
			So(bi.ProvisionableLabels["cros-something"], ShouldEqual, "hello")
			b.Close(ctx)
			So(bi.ProvisionableLabels, ShouldHaveLength, 2)
			So(bi.ProvisionableLabels["cros-version"], ShouldEqual, "hi")
			So(bi.ProvisionableLabels["cros-something"], ShouldEqual, "hello")
		})
		Convey("Remove provision label when host-info does not have it in scope of expected labels", func() {
			bi := &swmbot.LocalState{
				ProvisionableLabels: map[string]string{
					"cros-version":   "hi",
					"cros-something": "hello",
				},
				ProvisionableAttributes: make(map[string]string),
			}
			b := &Borrower{
				hostInfo: &hostinfo.HostInfo{
					Labels:     []string{},
					Attributes: make(map[string]string),
				},
				botInfo: bi,
			}
			So(bi.ProvisionableLabels, ShouldHaveLength, 2)
			So(bi.ProvisionableLabels["cros-version"], ShouldEqual, "hi")
			So(bi.ProvisionableLabels["cros-something"], ShouldEqual, "hello")
			b.Close(ctx)
			So(bi.ProvisionableLabels, ShouldHaveLength, 1)
			So(bi.ProvisionableLabels["cros-something"], ShouldEqual, "hello")
		})
	})
}
