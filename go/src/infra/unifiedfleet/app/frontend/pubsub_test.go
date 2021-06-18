// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/model/registration"
)

func TestUpdateAssetInfoHelper(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("Testing updateAssetInfoHelper", t, func() {
		Convey("Update non-existing asset", func() {
			// Shouldn't work as we didn't create the asset
			err := updateAssetInfoHelper(ctx, &ufspb.AssetInfo{
				AssetTag: "test-tag",
			})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Entity not found")
		})
		Convey("Update asset with missing machine", func() {
			a1 := &ufspb.Asset{
				Name:  "test-tag",
				Model: "test-model",
				Type:  ufspb.AssetType_DUT,
				Location: &ufspb.Location{
					Zone: ufspb.Zone_ZONE_CROS_GOOGLER_DESK,
					Rack: "test-rack",
				},
			}
			// Create an asset.
			_, err := registration.CreateAsset(ctx, a1)
			So(err, ShouldBeNil)
			// Update a dut asset without machine.
			err = updateAssetInfoHelper(ctx, &ufspb.AssetInfo{
				AssetTag:    "test-tag",
				Model:       "test-model",
				BuildTarget: "test-target",
			})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Entity not found")
		})
		Convey("Update asset info - Happy path", func() {
			l1 := &ufspb.Location{
				Zone: ufspb.Zone_ZONE_CROS_GOOGLER_DESK,
				Rack: "test-rack",
			}
			a1 := &ufspb.Asset{
				Name:     "test-tag1",
				Model:    "test-model",
				Type:     ufspb.AssetType_DUT,
				Location: l1,
			}
			m1 := &ufspb.Machine{
				Name: "test-tag1",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						Model: "test-model",
					},
				},
				Location: l1,
			}
			// Create an asset
			_, err := registration.CreateAsset(ctx, a1)
			So(err, ShouldBeNil)
			// Create a corresponding machine
			_, err = registration.CreateMachine(ctx, m1)
			So(err, ShouldBeNil)
			// Update buildtarget for the asset
			err = updateAssetInfoHelper(ctx, &ufspb.AssetInfo{
				AssetTag:    "test-tag1",
				Model:       "test-model",
				BuildTarget: "test-target",
			})
			So(err, ShouldBeNil)
			// Machine should reflect the change
			m2, err := registration.GetMachine(ctx, "test-tag1")
			So(m2.GetChromeosMachine().GetBuildTarget(), ShouldEqual, "test-target")
		})
		Convey("Update asset info - Machine avoids HWID, phase, sku and mac", func() {
			l1 := &ufspb.Location{
				Zone: ufspb.Zone_ZONE_CROS_GOOGLER_DESK,
				Rack: "test-rack",
			}
			a1 := &ufspb.Asset{
				Name:     "test-tag2",
				Model:    "test-model",
				Type:     ufspb.AssetType_DUT,
				Location: l1,
			}
			m1 := &ufspb.Machine{
				Name: "test-tag2",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						Model:      "test-model",
						MacAddress: "FF:FF:FF:EE:EE:EE",
						Sku:        "21",
						Phase:      "PVT",
						Hwid:       "TESTHWID 123FGHEASFG",
					},
				},
				Location: l1,
			}
			// Create an asset
			_, err := registration.CreateAsset(ctx, a1)
			So(err, ShouldBeNil)
			// Create a corresponding machine
			_, err = registration.CreateMachine(ctx, m1)
			So(err, ShouldBeNil)
			// Update buildtarget for the asset
			err = updateAssetInfoHelper(ctx, &ufspb.AssetInfo{
				AssetTag:           "test-tag2",
				Model:              "test-model",
				Sku:                "0",
				EthernetMacAddress: "11:11:11:22:22:22",
				Phase:              "EVT",
				Hwid:               "NOTEST HWID124RFGG",
			})
			So(err, ShouldBeNil)
			// Machine should not reflect the change
			m2, err := registration.GetMachine(ctx, "test-tag2")
			So(err, ShouldBeNil)
			So(m2.GetChromeosMachine().GetSku(), ShouldEqual, "21")
			So(m2.GetChromeosMachine().GetHwid(), ShouldEqual, "TESTHWID 123FGHEASFG")
			So(m2.GetChromeosMachine().GetPhase(), ShouldEqual, "PVT")
			So(m2.GetChromeosMachine().GetMacAddress(), ShouldEqual, "FF:FF:FF:EE:EE:EE")
			// Asset should record the change
			a2, err := registration.GetAsset(ctx, "test-tag2")
			So(err, ShouldBeNil)
			So(a2.GetInfo().GetPhase(), ShouldEqual, "EVT")
			So(a2.GetInfo().GetEthernetMacAddress(), ShouldEqual, "11:11:11:22:22:22")
			So(a2.GetInfo().GetHwid(), ShouldEqual, "NOTEST HWID124RFGG")
			So(a2.GetInfo().GetSku(), ShouldEqual, "0")
		})
	})
}
