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

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/cmd/skylab_swarming_worker/internal/swmbot"
	"infra/libs/skylab/inventory"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsutil "infra/unifiedfleet/app/util"
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

// Load loads the bot's DUT's info from the UFS.
//
// This function returns a Store that should be closed to update the inventory
// with any changes to the info, using a supplied UpdateFunc. If UpdateFunc is
// nil, the inventory is not updated.
func Load(ctx context.Context, b *swmbot.Info, f UpdateFunc) (*Store, error) {
	return load(ctx, b, f, getDutInfoFromUFS)
}

type getDutInfoFuncUFS func(context.Context, ufsAPI.FleetClient, *ufsAPI.GetChromeOSDeviceDataRequest) (*ufspb.ChromeOSDeviceData, error)

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

func loadFromUFS(ctx context.Context, b *swmbot.Info, gf getDutInfoFuncUFS) (*inventory.DeviceUnderTest, error) {
	client, err := swmbot.UFSClient(ctx, b)
	if err != nil {
		return nil, errors.Annotate(err, "load from UFS: initialize UFS client").Err()
	}
	req := ufsAPI.GetChromeOSDeviceDataRequest{
		ChromeosDeviceId: b.DUTID,
	}
	resp, err := gf(ctx, client, &req)
	if err != nil {
		return nil, errors.Annotate(err, "load from UFS").Err()
	}
	return resp.GetDutV1(), nil
}

func load(ctx context.Context, b *swmbot.Info, uf UpdateFunc, gfUFS getDutInfoFuncUFS) (*Store, error) {
	ctx, err := swmbot.WithSystemAccount(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "load DUT host info").Err()
	}
	log.Printf("Loading DUT info from UFS")
	dut, err := loadFromUFS(ctx, b, gfUFS)
	if err != nil {
		log.Printf("(not fatal) fail to load DUT from inventory V2: %s", err)
	}

	c, err := swmbot.InventoryClient(ctx, b)
	if err != nil {
		return nil, errors.Annotate(err, "load DUT host info").Err()
	}
	// TODO(gregorynisbet): should failure to get the stableversion information
	// cause the entire request to error out?
	hostname := dut.GetCommon().GetHostname()
	sv, err := getStableVersion(ctx, c, hostname)
	if err != nil {
		sv = map[string]string{}
		log.Printf("load: getting stable version: sv (%v) err (%v)", sv, err)
	}
	// once we reach this point, sv is guaranteed to be non-nil
	store := &Store{
		DUT:            dut,
		oldDUT:         proto.Clone(dut).(*inventory.DeviceUnderTest),
		updateFunc:     uf,
		StableVersions: sv,
	}
	return store, nil
}

func getDutInfoFromUFS(ctx context.Context, c ufsAPI.FleetClient, req *ufsAPI.GetChromeOSDeviceDataRequest) (*ufspb.ChromeOSDeviceData, error) {
	var resp *ufspb.ChromeOSDeviceData
	f := func() (err error) {
		osCtx := swmbot.SetupContext(ctx, ufsutil.OSNamespace)
		resp, err = c.GetChromeOSDeviceData(osCtx, req)
		return err
	}
	if err := retry.Retry(ctx, retry.Default, f, retry.LogCallback(ctx, "dutinfo.getDutInfoFuncUFS")); err != nil {
		return nil, errors.Annotate(err, "retry getDutInfoFuncUFS").Err()
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
