// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/external"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/util"
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

func mockDutLabelNoServer() *ufspb.DutLabel {
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
				Value: "test-sku-no-server",
			},
			{
				Name:  "variant",
				Value: "test-variant-no-server",
			},
		},
	}
}

func fakeUpdateHwidData(ctx context.Context, d *ufspb.DutLabel, hwid string, updatedTime time.Time) (*configuration.HwidDataEntity, error) {
	hwidData, err := proto.Marshal(d)
	if err != nil {
		return nil, errors.Annotate(err, "failed to marshal HwidData %s", d).Err()
	}

	if hwid == "" {
		return nil, status.Errorf(codes.Internal, "Empty hwid")
	}

	entity := &configuration.HwidDataEntity{
		ID:       hwid,
		HwidData: hwidData,
		Updated:  updatedTime,
	}

	if err := datastore.Put(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

func TestGetHwidDataV1(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	ctx = external.WithTestingContext(ctx)
	ctx = useTestingCfg(ctx)
	datastore.GetTestable(ctx).Consistent(true)

	es, err := external.GetServerInterface(ctx)
	if err != nil {
		t.Fatalf("Failed to get server interface: %s", err)
	}
	client, err := es.NewHwidClientInterface(ctx)
	if err != nil {
		t.Fatalf("Failed to get fake hwid client interface: %s", err)
	}

	t.Run("happy path - get cached data from datastore", func(t *testing.T) {
		// Server should respond but since cache is within range, cache should be
		// returned and not updated.
		id := "test"

		// Test if server is responding.
		serverRsp, err := client.QueryHwid(ctx, id)
		if err != nil {
			t.Fatalf("Fake hwid server responded with error: %s", err)
		}
		if diff := cmp.Diff(mockDutLabel(), serverRsp, protocmp.Transform()); diff != "" {
			t.Errorf("Fake hwid server returned unexpected diff (-want +got):\n%s", diff)
		}

		// Test getting data from datastore.
		cacheTime := time.Now().UTC().Add(-30 * time.Minute)
		_, err = fakeUpdateHwidData(ctx, mockDutLabel(), id, cacheTime)
		if err != nil {
			t.Fatalf("fakeUpdateHwidData failed: %s", err)
		}
		want := &ufspb.HwidData{
			Sku:     "test-sku",
			Variant: "test-variant",
		}
		got, err := GetHwidDataV1(ctx, client, id)
		if err != nil {
			t.Fatalf("GetHwidDataV1 failed: %s", err)
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("GetHwidDataV1 returned unexpected diff (-want +got):\n%s", diff)
		}
		hwidEnt, _ := configuration.GetHwidData(ctx, id)
		if diff := cmp.Diff(cacheTime, hwidEnt.Updated, cmpopts.EquateApproxTime(1*time.Millisecond)); diff != "" {
			t.Errorf("Cache time has unexpected diff (-want +got):\n%s", diff)
		}
		datastore.Delete(ctx, hwidEnt)
	})

	t.Run("get cached data from datastore; hwid server errors", func(t *testing.T) {
		// Server should respond nil so method should return last cached entity from
		// the datastore.
		id := "test-no-server"

		// Test if server is responding nil.
		serverRsp, err := client.QueryHwid(ctx, id)
		if err == nil {
			t.Fatalf("Fake hwid server responded without error")
		}
		if diff := cmp.Diff(&ufspb.DutLabel{}, serverRsp, protocmp.Transform()); diff != "" {
			t.Errorf("Fake hwid server returned unexpected diff (-want +got):\n%s", diff)
		}

		// Test getting data from datastore.
		hwidEnt, err := configuration.UpdateHwidData(ctx, mockDutLabelNoServer(), id)
		if err != nil {
			t.Fatalf("UpdateHwidData failed: %s", err)
		}
		want := &ufspb.HwidData{
			Sku:     "test-sku-no-server",
			Variant: "test-variant-no-server",
		}
		got, err := GetHwidDataV1(ctx, client, id)
		if err != nil {
			t.Fatalf("GetHwidDataV1 failed: %s", err)
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("GetHwidDataV1 returned unexpected diff (-want +got):\n%s", diff)
		}
		datastore.Delete(ctx, hwidEnt)
	})

	t.Run("empty datastore; get data from hwid server and update cache", func(t *testing.T) {
		// Datastore is empty so query hwid server. Server should respond with
		// DutLabel data and cache in datastore.
		id := "test"

		// No data should exist in datastore for id.
		_, err := configuration.GetHwidData(ctx, id)
		if err != nil && !util.IsNotFoundError(err) {
			t.Fatalf("Datastore already contains data for %s: %s", id, err)
		}

		// Test method and get data from server.
		want := &ufspb.HwidData{
			Sku:     "test-sku",
			Variant: "test-variant",
		}
		got, err := GetHwidDataV1(ctx, client, id)
		if err != nil {
			t.Fatalf("GetHwidDataV1 failed: %s", err)
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("GetHwidDataV1 returned unexpected diff (-want +got):\n%s", diff)
		}

		// Test if results were cached into datastore.
		hwidEnt, err := configuration.GetHwidData(ctx, id)
		if err != nil {
			if util.IsNotFoundError(err) {
				t.Fatalf("GetHwidDataV1 did not cache hwid server result")
			}
			t.Fatalf("GetHwidDataV1 unknown error: %s", err)
		}
		data, err := configuration.ParseHwidDataV1(hwidEnt)
		if err != nil {
			t.Fatalf("Failed to parse hwid data: %s", err)
		}
		if diff := cmp.Diff(want, data, protocmp.Transform()); diff != "" {
			t.Errorf("GetHwidDataV1 returned unexpected diff (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(time.Now().UTC(), hwidEnt.Updated, cmpopts.EquateApproxTime(2*time.Second)); diff != "" {
			t.Errorf("New cache time is outside margin of error; unexpected diff (-want +got):\n%s", diff)
		}
		datastore.Delete(ctx, hwidEnt)
	})

	t.Run("datastore data expired; update cache with hwid server", func(t *testing.T) {
		// Datastore data is expired so query hwid server. Server should respond
		// with DutLabel data and cache in datastore.
		id := "test"

		// Add expired data to datastore.
		expiredTime := time.Now().Add(-2 * time.Hour).UTC()
		fakeUpdateHwidData(ctx, mockDutLabelNoServer(), "test", expiredTime)
		want := &ufspb.HwidData{
			Sku:     "test-sku-no-server",
			Variant: "test-variant-no-server",
		}
		hwidEntExp, _ := configuration.GetHwidData(ctx, id)
		dataExp, _ := configuration.ParseHwidDataV1(hwidEntExp)
		if diff := cmp.Diff(want, dataExp, protocmp.Transform()); diff != "" {
			t.Errorf("GetHwidDataV1 returned unexpected diff (-want +got):\n%s", diff)
		}

		// Calling GetHwidDataV1 should immediately cache new data into datastore
		// and return the new data.
		want = &ufspb.HwidData{
			Sku:     "test-sku",
			Variant: "test-variant",
		}
		got, err := GetHwidDataV1(ctx, client, id)
		if err != nil {
			t.Fatalf("GetHwidDataV1 failed: %s", err)
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("GetHwidDataV1 returned unexpected diff (-want +got):\n%s", diff)
		}

		// Test if results were cached into datastore.
		hwidEnt, err := configuration.GetHwidData(ctx, id)
		if err != nil {
			t.Fatalf("GetHwidDataV1 unknown error: %s", err)
		}
		data, err := configuration.ParseHwidDataV1(hwidEnt)
		if err != nil {
			t.Fatalf("Failed to parse hwid data: %s", err)
		}
		if diff := cmp.Diff(want, data, protocmp.Transform()); diff != "" {
			t.Errorf("GetHwidDataV1 returned unexpected diff (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(time.Now().UTC(), hwidEnt.Updated, cmpopts.EquateApproxTime(2*time.Second)); diff != "" {
			t.Errorf("New cache time is outside margin of error; unexpected diff (-want +got):\n%s", diff)
		}
		datastore.Delete(ctx, hwidEnt)
	})

	t.Run("no data in datastore and hwid server errors", func(t *testing.T) {
		got, err := GetHwidDataV1(ctx, client, "test-err")
		if err != nil {
			t.Fatalf("GetHwidDataV1 unknown error: %s", err)
		}
		if got != nil {
			t.Errorf("GetHwidDataV1 is not nil: %s", got)
		}
	})

	t.Run("no data in datastore and throttle hwid server", func(t *testing.T) {
		cfgLst := &config.Config{
			HwidServiceTrafficRatio: 0,
		}
		trafficCtx := config.Use(ctx, cfgLst)

		got, err := GetHwidDataV1(trafficCtx, client, "test-no-data")
		if err != nil {
			t.Fatalf("GetHwidDataV1 unknown error: %s", err)
		}
		if got != nil {
			t.Errorf("GetHwidDataV1 is not nil: %s", got)
		}
	})
}
