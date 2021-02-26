// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package hostinfo

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"infra/cmd/skylab_swarming_worker/internal/swmbot"
	"infra/libs/skylab/autotest/hostinfo"
)

func TestCopyToProvisioningData(t *testing.T) {
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
			So(bi.ProvisionableAttributes, ShouldHaveLength, 0)
			b.Close(ctx)
			So(bi.ProvisionableLabels, ShouldHaveLength, 0)
			So(bi.ProvisionableAttributes, ShouldHaveLength, 0)
		})
		Convey("Set provisioning label and attributes when host-info provide info", func() {
			bi := &swmbot.LocalState{
				ProvisionableLabels:     make(map[string]string),
				ProvisionableAttributes: make(map[string]string),
			}
			b := &Borrower{
				hostInfo: &hostinfo.HostInfo{
					Labels: []string{"cros-version:hi"},
					Attributes: map[string]string{
						"job_repo_url": "some_package_url",
					},
				},
				botInfo: bi,
			}
			So(bi.ProvisionableLabels, ShouldHaveLength, 0)
			So(bi.ProvisionableAttributes, ShouldHaveLength, 0)
			b.Close(ctx)
			So(bi.ProvisionableLabels, ShouldHaveLength, 1)
			So(bi.ProvisionableLabels["cros-version"], ShouldEqual, "hi")
			So(bi.ProvisionableAttributes, ShouldHaveLength, 1)
			So(bi.ProvisionableAttributes["job_repo_url"], ShouldEqual, "some_package_url")
		})
		Convey("Update provisioning label and attributes when host-info provide info", func() {
			bi := &swmbot.LocalState{
				ProvisionableLabels: map[string]string{
					"cros-something": "hello",
					"cros-version":   "hello",
				},
				ProvisionableAttributes: map[string]string{
					"attr-something": "hello",
					"job_repo_url":   "hello",
				},
			}
			b := &Borrower{
				hostInfo: &hostinfo.HostInfo{
					Labels: []string{"cros-version:hi"},
					Attributes: map[string]string{
						"job_repo_url": "hi2",
					},
				},
				botInfo: bi,
			}
			So(bi.ProvisionableLabels, ShouldHaveLength, 2)
			So(bi.ProvisionableLabels["cros-version"], ShouldEqual, "hello")
			So(bi.ProvisionableLabels["cros-something"], ShouldEqual, "hello")
			So(bi.ProvisionableAttributes, ShouldHaveLength, 2)
			So(bi.ProvisionableAttributes["attr-something"], ShouldEqual, "hello")
			So(bi.ProvisionableAttributes["job_repo_url"], ShouldEqual, "hello")
			b.Close(ctx)
			So(bi.ProvisionableLabels, ShouldHaveLength, 2)
			So(bi.ProvisionableLabels["cros-version"], ShouldEqual, "hi")
			So(bi.ProvisionableLabels["cros-something"], ShouldEqual, "hello")
			So(bi.ProvisionableAttributes, ShouldHaveLength, 2)
			So(bi.ProvisionableAttributes["attr-something"], ShouldEqual, "hello")
			So(bi.ProvisionableAttributes["job_repo_url"], ShouldEqual, "hi2")
		})
		Convey("Remove provision label and attributes when host-info does not have it in scope of expected labels", func() {
			bi := &swmbot.LocalState{
				ProvisionableLabels: map[string]string{
					"cros-version":   "hi",
					"cros-something": "hello",
				},
				ProvisionableAttributes: map[string]string{
					"attr-something": "hello",
					"job_repo_url":   "hello",
				},
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
			So(bi.ProvisionableAttributes, ShouldHaveLength, 2)
			So(bi.ProvisionableAttributes["attr-something"], ShouldEqual, "hello")
			So(bi.ProvisionableAttributes["job_repo_url"], ShouldEqual, "hello")
			b.Close(ctx)
			So(bi.ProvisionableLabels, ShouldHaveLength, 1)
			So(bi.ProvisionableLabels["cros-something"], ShouldEqual, "hello")
			So(bi.ProvisionableAttributes, ShouldHaveLength, 1)
			So(bi.ProvisionableAttributes["attr-something"], ShouldEqual, "hello")
		})
	})
}
