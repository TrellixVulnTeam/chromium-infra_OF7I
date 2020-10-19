// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/genproto/protobuf/field_mask"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/registration"
)

func mockRack(name, row string, zone ufspb.Zone) *ufspb.Rack {
	return &ufspb.Rack{
		Name: name,
		Location: &ufspb.Location{
			Zone:        zone,
			Row:         row,
			BarcodeName: name,
		},
	}
}

func mockAsset(name, model, row, rack, pos, host string, aType ufspb.AssetType, zone ufspb.Zone) *ufspb.Asset {
	return &ufspb.Asset{
		Name:  name,
		Type:  aType,
		Model: model,
		Location: &ufspb.Location{
			Zone:        zone,
			Row:         row,
			Rack:        rack,
			Position:    pos,
			BarcodeName: host,
		},
	}
}

func mockAssetInfo(serialNumber, costCenter, googleCodeName, model, buildTarget, referenceBoard, ethernetMacAddress, sku, phase string) *ufspb.AssetInfo {
	return &ufspb.AssetInfo{
		SerialNumber:       serialNumber,
		CostCenter:         costCenter,
		GoogleCodeName:     googleCodeName,
		Model:              model,
		BuildTarget:        buildTarget,
		ReferenceBoard:     referenceBoard,
		EthernetMacAddress: ethernetMacAddress,
		Sku:                sku,
		Phase:              phase,
	}
}

func TestAssetRegistration(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("Testing AssetRegistration", t, func() {
		Convey("Register machine with existing rack", func() {
			r := mockRack("chromeos6-row2-rack3", "2", ufspb.Zone_ZONE_CHROMEOS6)
			_, err := registration.CreateRack(ctx, r)
			So(err, ShouldBeNil)
			a := mockAsset("C001001", "eve", "2", "chromeos6-row2-rack3", "1", "chromeos6-row2-rack3-host1", ufspb.AssetType_DUT, ufspb.Zone_ZONE_CHROMEOS6)
			_, err = AssetRegistration(ctx, a)
			So(err, ShouldBeNil)
		})
		Convey("Register machine with non-existent rack", func() {
			a := mockAsset("C001002", "eve", "2", "chromeos6-row3-rack3", "1", "chromeos6-row2-rack3-host1", ufspb.AssetType_DUT, ufspb.Zone_ZONE_CHROMEOS6)
			_, err := AssetRegistration(ctx, a)
			So(err, ShouldNotBeNil)
		})
		Convey("Register machine with invalid name", func() {
			r := mockRack("chromeos6-row4-rack3", "2", ufspb.Zone_ZONE_CHROMEOS6)
			_, err := registration.CreateRack(ctx, r)
			So(err, ShouldBeNil)
			a := mockAsset("", "eve", "4", "chromeos6-row4-rack3", "1", "chromeos6-row4-rack3-host1", ufspb.AssetType_DUT, ufspb.Zone_ZONE_CHROMEOS6)
			_, err = AssetRegistration(ctx, a)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestUpdateAsset(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("Testing UpdateAsset", t, func() {
		Convey("Update non existent asset", func() {
			r := mockRack("chromeos6-row2-rack3", "2", ufspb.Zone_ZONE_CHROMEOS6)
			_, err := registration.CreateRack(ctx, r)
			So(err, ShouldBeNil)
			b := mockAsset("C001001", "eve", "2", "chromeos6-row2-rack3", "1", "chromeos6-row2-rack3-host1", ufspb.AssetType_DUT, ufspb.Zone_ZONE_CHROMEOS6)
			_, err = UpdateAsset(ctx, b, &field_mask.FieldMask{Paths: []string{"type", "model"}})
			fmt.Println(err)
			So(err.Error(), ShouldContainSubstring, "unable to update asset C001001")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "assets/C001001")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Move asset to non-existent rack", func() {
			a := mockAsset("C001001", "eve", "2", "chromeos6-row2-rack3", "1", "chromeos6-row2-rack3-host1", ufspb.AssetType_DUT, ufspb.Zone_ZONE_CHROMEOS6)
			_, err := AssetRegistration(ctx, a)
			So(err, ShouldBeNil)
			b := mockAsset("C001001", "eve", "2", "chromeos6-row2-rack4", "1", "chromeos6-row2-rack3-host1", ufspb.AssetType_DUT, ufspb.Zone_ZONE_CHROMEOS6)
			_, err = UpdateAsset(ctx, b, &field_mask.FieldMask{Paths: []string{"location.rack"}})
			So(err.Error(), ShouldContainSubstring, "There is no Rack with RackID chromeos6-row2-rack4")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "assets/C001001")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Move Asset to existing rack", func() {
			r := mockRack("chromeos6-row2-rack5", "2", ufspb.Zone_ZONE_CHROMEOS6)
			_, err := registration.CreateRack(ctx, r)
			So(err, ShouldBeNil)
			r = mockRack("chromeos6-row2-rack6", "2", ufspb.Zone_ZONE_CHROMEOS6)
			_, err = registration.CreateRack(ctx, r)
			So(err, ShouldBeNil)
			a := mockAsset("C001002", "eve", "2", "chromeos6-row2-rack5", "1", "chromeos6-row2-rack3-host1", ufspb.AssetType_DUT, ufspb.Zone_ZONE_CHROMEOS6)
			_, err = AssetRegistration(ctx, a)
			So(err, ShouldBeNil)
			b := mockAsset("C001002", "eve", "2", "chromeos6-row2-rack6", "2", "chromeos6-row2-rack4-host2", ufspb.AssetType_DUT, ufspb.Zone_ZONE_CHROMEOS6)
			_, err = UpdateAsset(ctx, b, &field_mask.FieldMask{Paths: []string{"location.rack"}})
			So(err, ShouldBeNil)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "assets/C001002")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
		})

		Convey("Update Asset info of an asset", func() {
			r := mockRack("chromeos6-row2-rack7", "2", ufspb.Zone_ZONE_CHROMEOS6)
			_, err := registration.CreateRack(ctx, r)
			So(err, ShouldBeNil)
			a := mockAsset("C001003", "eve", "2", "chromeos6-row2-rack5", "3", "chromeos6-row2-rack3-host1", ufspb.AssetType_DUT, ufspb.Zone_ZONE_CHROMEOS6)
			_, err = AssetRegistration(ctx, a)
			So(err, ShouldBeNil)
			ai := mockAssetInfo("MTV100212", "", "", "", "", "", "", "", "")
			a.Info = ai
			_, err = UpdateAsset(ctx, a, &field_mask.FieldMask{Paths: []string{"info.serial_number"}})
			So(err, ShouldBeNil)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "assets/C001003")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "asset.info.serial_number")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, "MTV100212")
		})

		Convey("Update Asset with invalid mask", func() {
			a := mockAsset("C001004", "eve", "2", "chromeos6-row2-rack3", "1", "chromeos6-row2-rack3-host1", ufspb.AssetType_DUT, ufspb.Zone_ZONE_CHROMEOS6)
			_, err := AssetRegistration(ctx, a)
			So(err, ShouldBeNil)
			b := mockAsset("C001004", "eve", "2", "chromeos6-row2-rack3", "1", "chromeos6-row2-rack3-host1", ufspb.AssetType_DUT, ufspb.Zone_ZONE_CHROMEOS6)
			// Attempt to update name of the asset
			_, err = UpdateAsset(ctx, b, &field_mask.FieldMask{Paths: []string{"name"}})
			So(err, ShouldNotBeNil)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "assets/C001004")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			// Attempt to update name of the asset in asset info
			_, err = UpdateAsset(ctx, b, &field_mask.FieldMask{Paths: []string{"info.asset_tag"}})
			So(err, ShouldNotBeNil)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "assets/C001004")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			// Attempt to update timestamp of the asset
			_, err = UpdateAsset(ctx, b, &field_mask.FieldMask{Paths: []string{"update_time"}})
			So(err, ShouldNotBeNil)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "assets/C001004")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			// Attempt to clear zone of the asset
			b.Location.Zone = ufspb.Zone_ZONE_UNSPECIFIED
			_, err = UpdateAsset(ctx, b, &field_mask.FieldMask{Paths: []string{"location.zone"}})
			So(err, ShouldNotBeNil)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "assets/C001004")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			b.Location.Zone = ufspb.Zone_ZONE_CHROMEOS6
			// Attempt to clear rack of the asset
			b.Location.Rack = ""
			_, err = UpdateAsset(ctx, b, &field_mask.FieldMask{Paths: []string{"location.rack"}})
			So(err, ShouldNotBeNil)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "assets/C001004")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
	})
}
