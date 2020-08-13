// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package dutinfo implement loading Skylab DUT inventory info for the
// worker.
package dutinfo

import (
	"context"
	"fmt"
	"log"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/retry"

	invV2 "infra/appengine/cros/lab_inventory/api/v1"
	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/cmd/skylab_swarming_worker/internal/swmbot"
	"infra/libs/skylab/inventory"
)

// Store holds a DUT's inventory info and adds a Close method.
type Store struct {
	DUT            *inventory.DeviceUnderTest
	oldDUT         *inventory.DeviceUnderTest
	StableVersions map[string]string
	updateFunc     UpdateFunc
}

// Close updates the DUT's inventory info.  This method does nothing on
// subsequent calls.  This method is safe to call on a nil pointer.
func (s *Store) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if s.updateFunc == nil {
		return nil
	}
	c := s.DUT.GetCommon()
	new := c.GetLabels()
	inventory.SortLabels(new)
	old := s.oldDUT.GetCommon().GetLabels()
	inventory.SortLabels(old)
	if new.GetUselessSwitch() {
		*new.UselessSwitch = false
	}

	log.Printf("Calling label update function")
	if err := s.updateFunc(ctx, c.GetId(), s.oldDUT, s.DUT); err != nil {
		return errors.Annotate(err, "close DUT inventory").Err()
	}
	s.updateFunc = nil
	return nil
}

// UpdateFunc is used to implement inventory updating for any changes
// to the loaded DUT info.
type UpdateFunc func(ctx context.Context, dutID string, old *inventory.DeviceUnderTest, new *inventory.DeviceUnderTest) error

// Load loads the bot's DUT's info from the inventory V2.
//
// This function returns a Store that should be closed to update the inventory
// with any changes to the info, using a supplied UpdateFunc. If UpdateFunc is
// nil, the inventory is not updated.
func Load(ctx context.Context, b *swmbot.Info, f UpdateFunc) (*Store, error) {
	return load(ctx, b, f, getDutInfoFromV2)
}

type getDutInfoFuncV2 func(context.Context, invV2.InventoryClient, *invV2.GetCrosDevicesRequest) (*invV2.GetCrosDevicesResponse, error)

// getStableVersion fetches the current stable version from an inventory client
func getStableVersion(ctx context.Context, client fleet.InventoryClient, hostname string) (map[string]string, error) {
	log.Printf("getStableVersion: hostname (%s)", hostname)
	if hostname == "" {
		log.Printf("getStableVersion: failed validation for hostname")
		return nil, fmt.Errorf("getStableVersion: hostname cannot be \"\"")
	}
	req := &fleet.GetStableVersionRequest{
		Hostname: hostname,
	}
	log.Printf("getStableVersion: client request (%v) with retries", req)
	res, err := retryGetStableVersion(ctx, client, req)
	log.Printf("getStableVersion: client response (%v)", res)
	if err != nil {
		return nil, err
	}
	s := map[string]string{
		"cros":       res.GetCrosVersion(),
		"faft":       res.GetFaftVersion(),
		"firmware":   res.GetFirmwareVersion(),
		"servo-cros": res.GetServoCrosVersion(),
	}
	log.Printf("getStableVersion: stable version map (%v)", s)
	return s, nil
}

func loadFromV2(ctx context.Context, b *swmbot.Info, gf getDutInfoFuncV2) (*inventory.DeviceUnderTest, error) {
	client, err := swmbot.InventoryV2Client(ctx, b)
	if err != nil {
		return nil, errors.Annotate(err, "load from inventory V2: initialize V2 client").Err()
	}
	req := invV2.GetCrosDevicesRequest{
		Ids: []*invV2.DeviceID{
			{
				Id: &invV2.DeviceID_ChromeosDeviceId{
					ChromeosDeviceId: b.DUTID,
				},
			},
		},
	}
	resp, err := gf(ctx, client, &req)
	if err != nil {
		return nil, errors.Annotate(err, "load from inventory V2").Err()
	}
	if len(resp.GetData()) == 0 {
		return nil, errors.New("load from inventory V2: no results from V2 (neither successful nor failed)")
	}
	dut, err := invV2.AdaptToV1DutSpec(resp.GetData()[0])
	if err != nil {
		return nil, errors.Annotate(err, "load from inventory V2: fail to convert").Err()
	}
	return dut, nil
}

func load(ctx context.Context, b *swmbot.Info, uf UpdateFunc, gfV2 getDutInfoFuncV2) (*Store, error) {
	ctx, err := swmbot.WithSystemAccount(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "load DUT host info").Err()
	}
	log.Printf("Loading DUT info from Inventory V2")
	dutV2, err := loadFromV2(ctx, b, gfV2)
	if err != nil {
		log.Printf("(not fatal) fail to load DUT from inventory V2: %s", err)
	}

	c, err := swmbot.InventoryClient(ctx, b)
	if err != nil {
		return nil, errors.Annotate(err, "load DUT host info").Err()
	}
	// TODO(gregorynisbet): should failure to get the stableversion information
	// cause the entire request to error out?
	hostname := dutV2.GetCommon().GetHostname()
	sv, err := getStableVersion(ctx, c, hostname)
	if err != nil {
		sv = map[string]string{}
		log.Printf("load: getting stable version: sv (%v) err (%v)", sv, err)
	}
	// once we reach this point, sv is guaranteed to be non-nil
	store := &Store{
		DUT:            dutV2,
		oldDUT:         proto.Clone(dutV2).(*inventory.DeviceUnderTest),
		updateFunc:     uf,
		StableVersions: sv,
	}
	return store, nil
}

func getDutInfoFromV2(ctx context.Context, c invV2.InventoryClient, req *invV2.GetCrosDevicesRequest) (*invV2.GetCrosDevicesResponse, error) {
	var resp *invV2.GetCrosDevicesResponse
	f := func() (err error) {
		resp, err = c.GetCrosDevices(ctx, req)
		return err
	}
	if err := retry.Retry(ctx, retry.Default, f, retry.LogCallback(ctx, "dutinfo.getDutInfoFromV2")); err != nil {
		return nil, errors.Annotate(err, "retry getDutInfoFromV2").Err()
	}
	if len(resp.GetFailedDevices()) > 0 {
		fd := resp.GetFailedDevices()[0]
		return nil, errors.New(fmt.Sprintf("fail to load %s from V2: %s", fd.GetHostname(), fd.GetErrorMsg()))
	}
	return resp, nil
}

func retryGetStableVersion(ctx context.Context, client fleet.InventoryClient, req *fleet.GetStableVersionRequest) (*fleet.GetStableVersionResponse, error) {
	var resp *fleet.GetStableVersionResponse
	var err error
	f := func() error {
		resp, err = client.GetStableVersion(ctx, req)
		if err != nil {
			return err
		}
		return nil
	}
	if err := retry.Retry(ctx, retry.Default, f, retry.LogCallback(ctx, "dutinfo.retryGetStableVersion")); err != nil {
		return nil, errors.Annotate(err, "retry getStableVersion").Err()
	}
	return resp, nil
}
