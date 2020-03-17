// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"
	"math/rand"
	"time"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	api "infra/appengine/cros/lab_inventory/api/v1"
	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/app/frontend/internal/gitstore"
	"infra/libs/skylab/inventory"
)

type duoClient struct {
	gc *gitStoreClient
	ic *invServiceClient

	// A number in [0, 100] indicate the write traffic (deploy/update)
	// duplicated to inventory v2 service.
	writeTrafficRatio int
	// A number in [0, 100] indicate the read traffic fanning out to inventory
	// v2 service.
	readTrafficRatio int

	// The uuids of migration test devices.
	testingDeviceUUIDs stringset.Set

	// The uuids of migration test devices.
	testingDeviceNames stringset.Set

	// If we still write to v1.
	inventoryV2Only bool
}

func newDuoClient(ctx context.Context, gs *gitstore.InventoryStore, host string, readTrafficRatio, writeTrafficRatio int, testingUUIDs, testingNames []string, inventoryV2Only bool) (inventoryClient, error) {
	gc, err := newGitStoreClient(ctx, gs)
	if err != nil {
		return nil, errors.Annotate(err, "create git client").Err()
	}
	ic, err := newInvServiceClient(ctx, host)
	if err != nil {
		logging.Infof(ctx, "Failed to create inventory client of the duo client. Just return the git store client")
		return gc, nil
	}
	return &duoClient{
		gc:                 gc.(*gitStoreClient),
		ic:                 ic.(*invServiceClient),
		readTrafficRatio:   readTrafficRatio,
		writeTrafficRatio:  writeTrafficRatio,
		testingDeviceUUIDs: stringset.NewFromSlice(testingUUIDs...),
		testingDeviceNames: stringset.NewFromSlice(testingNames...),
		inventoryV2Only:    inventoryV2Only,
	}, nil
}

func (client *duoClient) willWriteToV2() bool {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Intn(100) < client.writeTrafficRatio
}

func (client *duoClient) willReadFromV2(req *fleet.GetDutInfoRequest) bool {
	if req.MustFromV1 {
		return false
	}
	if client.testingDeviceUUIDs.Has(req.GetId()) {
		return true
	}
	if client.testingDeviceNames.Has(req.GetHostname()) {
		return true
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Intn(100) < client.readTrafficRatio
}

func (client *duoClient) addManyDUTsToFleet(ctx context.Context, nds []*inventory.CommonDeviceSpecs, pickServoPort bool) (url string, ds []*inventory.CommonDeviceSpecs, err error) {
	if !client.inventoryV2Only {
		url, ds, err = client.gc.addManyDUTsToFleet(ctx, nds, pickServoPort)
		logging.Infof(ctx, "[v1] add dut result: %s, %s", url, err)
		logging.Infof(ctx, "[v1] spec returned: %s", ds)
	}
	if client.willWriteToV2() {
		url, ds, err = client.ic.addManyDUTsToFleet(ctx, nds, pickServoPort)
		logging.Infof(ctx, "[v2] add dut result: %s, %s", url, err)
		logging.Infof(ctx, "[v2] spec returned: %s", ds)
	}
	return
}

func (client *duoClient) getAssetsFromRegistration(ctx context.Context, assetIDList *api.AssetIDList) (*api.AssetResponse, error) {
	ds, err := client.ic.getAssetsFromRegistration(ctx, assetIDList)
	logging.Infof(ctx, "[v2] getAssetsFromRegistration assetResponse returned: %s, %s", ds, err)
	return ds, err
}

func (client *duoClient) updateAssetsInRegistration(ctx context.Context, assetList *api.AssetList) (*api.AssetResponse, error) {
	ds, err := client.ic.updateAssetsInRegistration(ctx, assetList)
	logging.Infof(ctx, "[v2] updateAssetsInRegistration assetResponse returned: %s, %s", ds, err)
	return ds, err
}

func (client *duoClient) updateDUTSpecs(ctx context.Context, od, nd *inventory.CommonDeviceSpecs, pickServoPort bool) (url string, err error) {
	if !client.inventoryV2Only {
		url, err = client.gc.updateDUTSpecs(ctx, od, nd, pickServoPort)
		logging.Infof(ctx, "[v1] update dut result: %s, %s", url, err)
	}
	if client.willWriteToV2() {
		url, err = client.ic.updateDUTSpecs(ctx, od, nd, pickServoPort)
		logging.Infof(ctx, "[v2] update dut result: %s, %s", url, err)
	}
	return
}

func (client *duoClient) deleteDUTsFromFleet(ctx context.Context, ids []string) (url string, deletedIds []string, err error) {
	if !client.inventoryV2Only {
		url, deletedIds, err = client.gc.deleteDUTsFromFleet(ctx, ids)
		logging.Infof(ctx, "[v1] delete dut result: %s, %s, %s", url, deletedIds, err)
	}
	if client.willWriteToV2() {
		url, deletedIds, err = client.ic.deleteDUTsFromFleet(ctx, ids)
		logging.Infof(ctx, "[v2] delete dut result: %s, %s, %s", url, deletedIds, err)
	}
	return
}

func (client *duoClient) selectDutsFromInventory(ctx context.Context, sel *fleet.DutSelector) (duts []*inventory.DeviceUnderTest, err error) {
	if !client.inventoryV2Only {
		duts, err = client.gc.selectDutsFromInventory(ctx, sel)
	}
	if client.willWriteToV2() {
		duts, err = client.ic.selectDutsFromInventory(ctx, sel)
		logging.Infof(ctx, "[v2] select duts by %v", sel)
		if len(duts) > 0 {
			logging.Infof(ctx, "[v2] selecting returns '%s'...(total %d duts)", duts[0].GetCommon().GetHostname(), len(duts))
		} else {
			logging.Infof(ctx, "[v2] selecting returns 0 duts")
		}
	}
	return
}

func (client *duoClient) commitBalancePoolChanges(ctx context.Context, changes []*fleet.PoolChange) (u string, err error) {
	if !client.inventoryV2Only {
		u, err = client.gc.commitBalancePoolChanges(ctx, changes)
	}
	if client.willWriteToV2() {
		u, err = client.ic.commitBalancePoolChanges(ctx, changes)
		logging.Infof(ctx, "[v2] Commit balancing pool result: %s: %s", u, err)
	}
	return
}

func (client *duoClient) getDutInfo(ctx context.Context, req *fleet.GetDutInfoRequest) ([]byte, time.Time, error) {
	if client.willReadFromV2(req) {
		dut, now, err := client.ic.getDutInfo(ctx, req)
		logging.Infof(ctx, "[v2] GetDutInfo result: %#v: %s", req, err)
		return dut, now, err
	}
	return client.gc.getDutInfo(ctx, req)
}
