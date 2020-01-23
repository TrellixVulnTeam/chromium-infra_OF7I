// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/retry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/app/frontend/internal/gitstore"
	"infra/libs/skylab/inventory"
)

type inventoryClient interface {
	addManyDUTsToFleet(context.Context, []*inventory.CommonDeviceSpecs, bool) (string, []*inventory.CommonDeviceSpecs, error)
	updateDUTSpecs(context.Context, *inventory.CommonDeviceSpecs, *inventory.CommonDeviceSpecs, bool) (string, error)
	deleteDUTsFromFleet(context.Context, []string) (string, []string, error)
	selectDutsFromInventory(context.Context, *fleet.DutSelector) ([]*inventory.DeviceUnderTest, error)
	commitBalancePoolChanges(context.Context, []*fleet.PoolChange) (string, error)
}

type gitStoreClient struct {
	store *gitstore.InventoryStore
}

func newGitStoreClient(ctx context.Context, gs *gitstore.InventoryStore) (inventoryClient, error) {
	return &gitStoreClient{
		store: gs,
	}, nil
}

func (client *gitStoreClient) addManyDUTsToFleet(ctx context.Context, nds []*inventory.CommonDeviceSpecs, pickServoPort bool) (string, []*inventory.CommonDeviceSpecs, error) {
	return addManyDUTsToFleet(ctx, client.store, nds, pickServoPort)
}

func (client *gitStoreClient) updateDUTSpecs(ctx context.Context, od, nd *inventory.CommonDeviceSpecs, pickServoPort bool) (string, error) {
	return updateDUTSpecs(ctx, client.store, od, nd, pickServoPort)
}

func (client *gitStoreClient) deleteDUTsFromFleet(ctx context.Context, ids []string) (string, []string, error) {
	return deleteDUTsFromFleet(ctx, client.store, ids)
}

func (client *gitStoreClient) selectDutsFromInventory(ctx context.Context, sel *fleet.DutSelector) ([]*inventory.DeviceUnderTest, error) {
	return selectDutsFromInventory(ctx, client.store, sel)
}

func (client *gitStoreClient) commitBalancePoolChanges(ctx context.Context, changes []*fleet.PoolChange) (string, error) {
	return commitBalancePoolChanges(ctx, client.store, changes)
}

func addManyDUTsToFleet(ctx context.Context, s *gitstore.InventoryStore, nds []*inventory.CommonDeviceSpecs, pickServoPort bool) (string, []*inventory.CommonDeviceSpecs, error) {
	var respURL string
	newDeviceToID := make(map[*inventory.CommonDeviceSpecs]string)

	f := func() error {
		var ds []*inventory.CommonDeviceSpecs

		for _, nd := range nds {
			ds = append(ds, proto.Clone(nd).(*inventory.CommonDeviceSpecs))
		}

		if err := s.Refresh(ctx); err != nil {
			return errors.Annotate(err, "add dut to fleet").Err()
		}

		// New cache after refreshing store.
		c := newGlobalInvCache(ctx, s)

		for i, d := range ds {
			hostname := d.GetHostname()
			logging.Infof(ctx, "add device to fleet: %s", hostname)
			if _, ok := c.hostnameToID[hostname]; ok {
				logging.Infof(ctx, "dut with hostname %s already exists, skip adding", hostname)
				continue
			}
			if pickServoPort && !hasServoPortAttribute(d) {
				if err := assignNewServoPort(s.Lab.Duts, d); err != nil {
					logging.Infof(ctx, "fail to assign new servo port, skip adding")
					continue
				}
			}
			nid := addDUTToStore(s, d)
			newDeviceToID[nds[i]] = nid
		}

		// TODO(ayatane): Implement this better than just regenerating the cache.
		c = newGlobalInvCache(ctx, s)

		for _, id := range newDeviceToID {
			if _, err := assignDUT(ctx, c, id); err != nil {
				return errors.Annotate(err, "add dut to fleet").Err()
			}
		}

		firstHostname := "<empty>"
		if len(ds) > 0 {
			firstHostname = ds[0].GetHostname()
		}

		url, err := s.Commit(ctx, fmt.Sprintf("Add %d new DUT(s) : %s ...", len(ds), firstHostname))
		if err != nil {
			return errors.Annotate(err, "add dut to fleet").Err()
		}

		respURL = url
		for _, nd := range nds {
			id := newDeviceToID[nd]
			nd.Id = &id
		}
		return nil
	}

	err := retry.Retry(ctx, transientErrorRetries(), f, retry.LogCallback(ctx, "addManyDUTsToFleet"))

	newDevices := make([]*inventory.CommonDeviceSpecs, 0)
	for nd := range newDeviceToID {
		newDevices = append(newDevices, nd)
	}
	return respURL, newDevices, err
}

// updateDUTSpecs updates the DUT specs for an existing DUT in the inventory.
func updateDUTSpecs(ctx context.Context, s *gitstore.InventoryStore, od, nd *inventory.CommonDeviceSpecs, pickServoPort bool) (string, error) {
	var respURL string
	f := func() error {
		// Clone device specs before modifications so that changes don't leak
		// across retries.
		d := proto.Clone(nd).(*inventory.CommonDeviceSpecs)

		if err := s.Refresh(ctx); err != nil {
			return errors.Annotate(err, "add new dut to inventory").Err()
		}

		if pickServoPort && !hasServoPortAttribute(d) {
			if err := assignNewServoPort(s.Lab.Duts, d); err != nil {
				return errors.Annotate(err, "add dut to fleet").Err()
			}
		}

		dut, exists := getDUTByID(s.Lab, od.GetId())
		if !exists {
			return status.Errorf(codes.NotFound, "no DUT with ID %s", od.GetId())
		}
		// TODO(crbug/929776) DUTs under deployment are not marked specially in the
		// inventory yet. This causes two problems:
		// - Another admin task (say repair) may get scheduled on the new bot
		//   before the deploy task we create.
		// - If the deploy task fails, the DUT will still enter the fleet, but may
		//   not be ready for use.
		if !proto.Equal(dut.GetCommon(), od) {
			return errors.Reason("DUT specs update conflict").Err()
		}
		dut.Common = d

		url, err := s.Commit(ctx, fmt.Sprintf("Update DUT %s", od.GetId()))
		if err != nil {
			return errors.Annotate(err, "update DUT specs").Err()
		}

		respURL = url
		return nil
	}
	err := retry.Retry(ctx, transientErrorRetries(), f, retry.LogCallback(ctx, "updateDUTSpecs"))
	return respURL, err
}

func deleteDUTsFromFleet(ctx context.Context, s *gitstore.InventoryStore, ids []string) (string, []string, error) {
	var changeURL string
	var removedIDs []string
	f := func() error {
		if err2 := s.Refresh(ctx); err2 != nil {
			return err2
		}
		removedDUTs := removeDUTWithHostnames(s, ids)
		url, err2 := s.Commit(ctx, fmt.Sprintf("delete %d duts", len(removedDUTs)))
		if gitstore.IsEmptyErr(err2) {
			return nil
		}
		if err2 != nil {
			return err2
		}

		// Captured variables only on success, hence at most once.
		changeURL = url
		removedIDs = make([]string, 0, len(removedDUTs))
		for _, d := range removedDUTs {
			removedIDs = append(removedIDs, d.GetCommon().GetId())
		}
		return nil
	}
	err := retry.Retry(ctx, transientErrorRetries(), f, retry.LogCallback(ctx, "DeleteDut"))
	return changeURL, removedIDs, err
}

func selectDutsFromInventory(ctx context.Context, store *gitstore.InventoryStore, sel *fleet.DutSelector) ([]*inventory.DeviceUnderTest, error) {
	if err := store.Refresh(ctx); err != nil {
		return nil, err
	}
	duts, err := GetDutsByEnvironment(ctx, store)
	if err != nil {
		return nil, nil
	}
	var filteredDuts []*inventory.DeviceUnderTest
	for _, d := range duts {
		if sel == nil || dutMatchesSelector(d, sel) {
			filteredDuts = append(filteredDuts, d)
		}
	}
	return filteredDuts, nil
}

func commitBalancePoolChanges(ctx context.Context, store *gitstore.InventoryStore, changes []*fleet.PoolChange) (string, error) {
	if len(changes) == 0 {
		// No inventory changes are required.
		// TODO(pprabhu) add a unittest enforcing this.
		return "", nil
	}
	if err := applyChanges(ctx, store.Lab, changes); err != nil {
		return "", errors.Annotate(err, "apply balance pool changes").Err()
	}
	return store.Commit(ctx, "balance pool")
}
