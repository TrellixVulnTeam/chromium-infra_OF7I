// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package dronecfg implements datastore access for storing drone
// configs.
package dronecfg

import (
	"context"
	"sort"

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
)

// DUT describes a DUT for the purpose of a drone config.
type DUT struct {
	ID       string
	Hostname string
}

// Entity is a drone config datastore entity.
type Entity struct {
	_kind    string `gae:"$kind,droneConfig"`
	Hostname string `gae:"$id"`
	DUTs     []DUT  `gae:",noindex"`
}

func modifyDUTs(self []DUT, operator func(stringset.Set, stringset.Set) stringset.Set, other []DUT) []DUT {
	dutMap := map[string]DUT{}
	for _, d := range append(self, other...) {
		dutMap[d.ID] = d
	}
	getIds := func(duts []DUT) []string {
		var result []string
		for _, d := range duts {
			result = append(result, d.ID)
		}
		return result
	}
	resultIds := operator(stringset.NewFromSlice(getIds(self)...), stringset.NewFromSlice(getIds(other)...)).ToSlice()
	sort.Strings(resultIds)
	var result []DUT
	for _, id := range resultIds {
		result = append(result, dutMap[id])
	}
	return result
}

// MergeDutsToDrones merge the drone config with the newly added DUTs and/or
// DUTs to be removed.
func MergeDutsToDrones(ctx context.Context, dronesToAddDut []Entity, dronesToRemoveDut []Entity) error {
	err := datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		// Make a drone list and a name map.
		dronesMap := map[string]*Entity{}
		var drones []*Entity
		for _, d := range append(dronesToAddDut, dronesToRemoveDut...) {
			if _, ok := dronesMap[d.Hostname]; !ok {
				e := &Entity{Hostname: d.Hostname}
				dronesMap[d.Hostname] = e
				drones = append(drones, e)
			}
		}

		if err := datastore.Get(ctx, drones); err != nil && !datastore.IsErrNoSuchEntity(err) {
			return err
		}
		for _, d := range dronesToAddDut {
			dronesMap[d.Hostname].DUTs = modifyDUTs(dronesMap[d.Hostname].DUTs, stringset.Set.Union, d.DUTs)
		}
		for _, d := range dronesToRemoveDut {
			dronesMap[d.Hostname].DUTs = modifyDUTs(dronesMap[d.Hostname].DUTs, stringset.Set.Difference, d.DUTs)
		}
		// Keep drones with 0 DUTs in datastore.
		if err := datastore.Put(ctx, drones); err != nil {
			return err
		}
		return nil
	}, nil)
	if err != nil {
		return errors.Annotate(err, "merge drone configs").Err()
	}
	return nil
}

// Get gets a drone config from datastore by hostname.
func Get(ctx context.Context, hostname string) (Entity, error) {
	e := Entity{Hostname: hostname}
	if err := datastore.Get(ctx, &e); err != nil {
		return e, errors.Annotate(err, "get drone config").Err()
	}
	return e, nil
}

const queenDronePrefix = "drone-queen-"

// QueenDroneName returns the name of the fake drone whose DUTs should
// be pushed to the drone queen service.
func QueenDroneName(env string) string {
	return queenDronePrefix + env
}
