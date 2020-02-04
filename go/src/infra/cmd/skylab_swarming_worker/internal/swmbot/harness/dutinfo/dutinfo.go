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
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/kylelemons/godebug/pretty"
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
func (s *Store) Close() error {
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
	if err := s.updateFunc(c.GetId(), s.oldDUT, s.DUT); err != nil {
		return errors.Annotate(err, "close DUT inventory").Err()
	}
	s.updateFunc = nil
	return nil
}

// UpdateFunc is used to implement inventory updating for any changes
// to the loaded DUT info.
type UpdateFunc func(dutID string, old *inventory.DeviceUnderTest, new *inventory.DeviceUnderTest) error

// LoadCached loads the bot's DUT's info from the inventory. Returned inventory
// data may be slightly stale compared to the source of truth of the inventory.
//
// This function returns a Store that should be closed to update the inventory
// with any changes to the info, using a supplied UpdateFunc.  If UpdateFunc is
// nil, the inventory is not updated.
func LoadCached(ctx context.Context, b *swmbot.Info, f UpdateFunc) (*Store, error) {
	return load(ctx, b, f, getCached, getDutInfoFromV2)
}

// LoadFresh loads the bot's DUT's info from the inventory. Returned inventory
// data is guaranteed to be up-to-date with the source of truth of the
// inventory. This function may take longer than LoadCached because it needs
// to wait for the caches to be updated.
//
// This function returns a Store that should be closed to update the inventory
// with any changes to the info, using a supplied UpdateFunc.  If UpdateFunc is
// nil, the inventory is not updated.
func LoadFresh(ctx context.Context, b *swmbot.Info, f UpdateFunc) (*Store, error) {
	return load(ctx, b, f, getUncached, getDutInfoFromV2)
}

type getDutInfoFunc func(context.Context, fleet.InventoryClient, *fleet.GetDutInfoRequest) (*fleet.GetDutInfoResponse, error)

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
	log.Printf("getStableVersion: client request (%v)", req)
	res, err := client.GetStableVersion(ctx, req)
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

func load(ctx context.Context, b *swmbot.Info, uf UpdateFunc, gf getDutInfoFunc, gfV2 getDutInfoFuncV2) (*Store, error) {
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
	resp, err := gf(ctx, c, &fleet.GetDutInfoRequest{Id: b.DUTID})
	if err != nil {
		return nil, errors.Annotate(err, "load DUT host info").Err()
	}
	var d inventory.DeviceUnderTest
	if err := proto.Unmarshal(resp.Spec, &d); err != nil {
		return nil, errors.Annotate(err, "load DUT host info").Err()
	}

	log.Printf("Comparison between V1 & V2: \n%s", pretty.Compare(dutV2, &d))
	// TODO(gregorynisbet): should failure to get the stableversion information
	// cause the entire request to error out?
	hostname := d.GetCommon().GetHostname()
	sv, err := getStableVersion(ctx, c, hostname)
	if err != nil {
		sv = map[string]string{}
		log.Printf("load: getting stable version: sv (%v) err (%v)", sv, err)
	}
	// once we reach this point, sv is guaranteed to be non-nil
	store := &Store{
		DUT:            &d,
		oldDUT:         proto.Clone(&d).(*inventory.DeviceUnderTest),
		updateFunc:     uf,
		StableVersions: sv,
	}
	return store, nil
}

func getDutInfoFromV2(ctx context.Context, c invV2.InventoryClient, req *invV2.GetCrosDevicesRequest) (*invV2.GetCrosDevicesResponse, error) {
	resp, err := c.GetCrosDevices(ctx, req)
	if err != nil {
		return nil, errors.Annotate(err, "get dut info from inventory V2").Err()
	}
	if len(resp.GetFailedDevices()) > 0 {
		f := resp.GetFailedDevices()[0]
		return nil, errors.New(fmt.Sprintf("fail to load %s from V2: %s", f.GetHostname(), f.GetErrorMsg()))
	}
	return resp, nil
}

// getCached obtains DUT info from the inventory service ignoring cache
// freshness.
func getCached(ctx context.Context, c fleet.InventoryClient, req *fleet.GetDutInfoRequest) (*fleet.GetDutInfoResponse, error) {
	resp, err := c.GetDutInfo(ctx, req)
	if err != nil {
		return nil, errors.Annotate(err, "get cached").Err()
	}
	return resp, nil
}

// getUncached obtains DUT info from the inventory service ensuring that
// returned info is up-to-date with the source of truth.
func getUncached(ctx context.Context, c fleet.InventoryClient, req *fleet.GetDutInfoRequest) (*fleet.GetDutInfoResponse, error) {
	var resp *fleet.GetDutInfoResponse
	start := time.Now().UTC()
	f := func() error {
		iresp, err := getCached(ctx, c, req)
		if err != nil {
			return err
		}
		if err := ensureResponseUpdatedSince(iresp, start); err != nil {
			return err
		}

		// Only update captured variables on success.
		resp = iresp
		return nil
	}

	if err := retry.Retry(ctx, cacheRefreshRetryFactory, f, retry.LogCallback(ctx, "dutinfo.getCached")); err != nil {
		return nil, errors.Annotate(err, "get uncached").Err()
	}
	return resp, nil
}

// cacheRefreshRetryFactory is a retry.Factory to configure retries to wait for
// inventory cache to be refreshed.
func cacheRefreshRetryFactory() retry.Iterator {
	// Cache is refreshed via a cron task that runs every minute or so.
	// Retry at: 10s, 30s, 70s, 2m10s, 3m10s, 4m10s, 5m10s
	return &retry.ExponentialBackoff{
		Limited: retry.Limited{
			Delay: 10 * time.Second,
			// Leave a little headroom for the last retry at 5m10s.
			MaxTotal: 5*time.Minute + 20*time.Second,
			// We enforce limit via MaxTotal
			Retries: -1,
		},
		MaxDelay: 1 * time.Minute,
	}
}

func ensureResponseUpdatedSince(r *fleet.GetDutInfoResponse, t time.Time) error {
	if r.Updated == nil {
		return errors.Reason("ensure uncached response: updated field is nil").Err()
	}
	u, err := ptypes.Timestamp(r.Updated)
	if err != nil {
		return errors.Annotate(err, "ensure uncached response").Err()
	}
	if t.After(u) {
		return errors.Reason("ensure uncached response: last update %s before start", t.Sub(u).String()).Err()
	}
	return nil
}
