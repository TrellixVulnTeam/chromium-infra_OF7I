// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	ufspb "infra/unifiedfleet/api/v1/proto"
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
