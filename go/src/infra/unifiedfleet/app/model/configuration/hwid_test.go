// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"

	ufspb "infra/unifiedfleet/api/v1/models"
)

func mockDutLabel() *ufspb.DutLabel {
	return &ufspb.DutLabel{
		PossibleLabels: []string{
			"test-possible-1",
			"test-possible-2",
		},
		Labels: []*ufspb.DutLabel_Label{
			{
				Name:  "test-label-1",
				Value: "test-value-1",
			},
			{
				Name:  "Sku",
				Value: "test-sku",
			},
			{
				Name:  "variant",
				Value: "test-variant",
			},
		},
	}
}

func TestUpdateHwidData(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)

	t.Run("update non-existent HwidData", func(t *testing.T) {
		want := mockDutLabel()
		got, err := UpdateHwidData(ctx, want, "test-hwid")
		if err != nil {
			t.Fatalf("UpdateHwidData failed: %s", err)
		}
		gotProto, err := got.GetProto()
		if err != nil {
			t.Fatalf("GetProto failed: %s", err)
		}
		if diff := cmp.Diff(want, gotProto, protocmp.Transform()); diff != "" {
			t.Errorf("UpdateHwidData returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("update existent HwidData", func(t *testing.T) {
		hd2Id := "test-hwid-2"
		hd2 := mockDutLabel()

		hd2update := mockDutLabel()
		hd2update.PossibleLabels = append(hd2update.PossibleLabels, "test-possible-3")

		// Insert hd2 into datastore
		_, _ = UpdateHwidData(ctx, hd2, hd2Id)

		// Update hd2
		got, err := UpdateHwidData(ctx, hd2update, hd2Id)
		if err != nil {
			t.Fatalf("UpdateHwidData failed: %s", err)
		}
		gotProto, err := got.GetProto()
		if err != nil {
			t.Fatalf("GetProto failed: %s", err)
		}
		if diff := cmp.Diff(hd2update, gotProto, protocmp.Transform()); diff != "" {
			t.Errorf("UpdateHwidData returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("update HwidData with empty hwid", func(t *testing.T) {
		hd3 := mockDutLabel()
		got, err := UpdateHwidData(ctx, hd3, "")
		if err == nil {
			t.Errorf("UpdateHwidData succeeded with empty hwid")
		}
		if c := status.Code(err); c != codes.Internal {
			t.Errorf("Unexpected error when calling UpdateHwidData: %s", err)
		}
		var hdNil *HwidDataEntity = nil
		if diff := cmp.Diff(hdNil, got); diff != "" {
			t.Errorf("UpdateHwidData returned unexpected diff (-want +got):\n%s", diff)
		}
	})
}

func TestGetHwidData(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)

	t.Run("get HwidData by existing ID", func(t *testing.T) {
		id := "test-hwid"
		want := mockDutLabel()
		_, err := UpdateHwidData(ctx, want, id)
		if err != nil {
			t.Fatalf("UpdateHwidData failed: %s", err)
		}

		got, err := GetHwidData(ctx, id)
		if err != nil {
			t.Fatalf("GetHwidData failed: %s", err)
		}
		gotProto, err := got.GetProto()
		if err != nil {
			t.Fatalf("GetProto failed: %s", err)
		}
		if diff := cmp.Diff(want, gotProto, protocmp.Transform()); diff != "" {
			t.Errorf("GetHwidData returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("get HwidData by non-existent ID", func(t *testing.T) {
		id := "test-hwid-2"
		_, err := GetHwidData(ctx, id)
		if err == nil {
			t.Errorf("GetHwidData succeeded with non-existent ID: %s", id)
		}
		if c := status.Code(err); c != codes.NotFound {
			t.Errorf("Unexpected error when calling GetHwidData: %s", err)
		}
	})
}

func TestParseHwidDataV1(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)

	id := "test-hwid"
	_, err := UpdateHwidData(ctx, mockDutLabel(), id)
	if err != nil {
		t.Fatalf("UpdateHwidData failed: %s", err)
	}

	t.Run("parse nil HwidEntity", func(t *testing.T) {
		var want *ufspb.HwidData = nil
		got, err := ParseHwidDataV1(nil)
		if err != nil {
			t.Fatalf("ParseHwidDataV1 failed: %s", err)
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("ParseHwidDataV1 returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("parse hwid data from HwidEntity", func(t *testing.T) {
		want := &ufspb.HwidData{
			Sku:     "test-sku",
			Variant: "test-variant",
		}
		ent, err := GetHwidData(ctx, id)
		if err != nil {
			t.Fatalf("GetHwidData failed: %s", err)
		}
		got, err := ParseHwidDataV1(ent)
		if err != nil {
			t.Fatalf("ParseHwidDataV1 failed: %s", err)
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("ParseHwidDataV1 returned unexpected diff (-want +got):\n%s", diff)
		}
	})
}
