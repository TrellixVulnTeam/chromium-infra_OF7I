// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dronecfg

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
)

func TestDroneConfig(t *testing.T) {

	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")

	Convey("Merge DUT to a new drone", t, func() {
		droneName := "drone.name"
		dutName := "dut1.name"
		dut2Name := "dut2.name"
		e := Entity{
			Hostname: droneName,
			DUTs:     []DUT{{Hostname: dutName, ID: "1"}},
		}

		err := MergeDutsToDrones(ctx, []Entity{e}, nil)
		if err != nil {
			t.Fatal(err)
		}
		datastore.GetTestable(ctx).Consistent(true)
		cfg, err := Get(ctx, droneName)
		if err != nil {
			t.Fatal(err)
		}

		So(cfg.DUTs, ShouldHaveLength, 1)
		So(cfg.DUTs[0].Hostname, ShouldEqual, dutName)

		Convey("Merge a new DUT to the dorne", func() {
			e := Entity{
				Hostname: droneName,
				DUTs:     []DUT{{Hostname: dut2Name, ID: "2"}},
			}
			err := MergeDutsToDrones(ctx, []Entity{e}, nil)
			if err != nil {
				t.Fatal(err)
			}
			cfg, err := Get(ctx, droneName)
			if err != nil {
				t.Fatal(err)
			}
			So(cfg.DUTs, ShouldHaveLength, 2)
			So(cfg.DUTs[0].Hostname, ShouldEqual, dutName)
			So(cfg.DUTs[1].Hostname, ShouldEqual, dut2Name)

			Convey("Remove a DUT from the dorne", func() {
				e := Entity{
					Hostname: droneName,
					DUTs:     []DUT{{Hostname: dutName, ID: "1"}},
				}
				err := MergeDutsToDrones(ctx, nil, []Entity{e})
				if err != nil {
					t.Fatal(err)
				}
				cfg, err := Get(ctx, droneName)
				if err != nil {
					t.Fatal(err)
				}
				So(cfg.DUTs, ShouldHaveLength, 1)
				So(cfg.DUTs[0].Hostname, ShouldEqual, dut2Name)

				Convey("Rename a DUT", func() {
					newName := "new.dut.name"
					e := Entity{
						Hostname: droneName,
						DUTs:     []DUT{{Hostname: newName, ID: "2"}},
					}
					err := MergeDutsToDrones(ctx, []Entity{e}, nil)
					if err != nil {
						t.Fatal(err)
					}
					cfg, err := Get(ctx, droneName)
					if err != nil {
						t.Fatal(err)
					}
					So(cfg.DUTs, ShouldHaveLength, 1)
					So(cfg.DUTs[0].Hostname, ShouldEqual, newName)
				})
			})
		})
	})
}
