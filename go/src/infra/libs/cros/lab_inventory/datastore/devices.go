// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package datastore contains datastore-related logic.
package datastore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"

	"infra/libs/cros/lab_inventory/utils"
)

// UUIDPrefix is the prefix we used to identify the system generated ID.
const UUIDPrefix = "UUID"

// A query in transaction requires to have Ancestor filter, see
// https://cloud.google.com/appengine/docs/standard/python/datastore/query-restrictions#queries_inside_transactions_must_include_ancestor_filters
func fakeAcestorKey(ctx context.Context) *datastore.Key {
	return datastore.MakeKey(ctx, DeviceKind, "key")
}

func addMissingID(devices []*lab.ChromeOSDevice) {
	// Use uuid as the device ID if asset id is not present.
	for _, d := range devices {
		if d.GetId().GetValue() == "" {
			d.Id.Value = fmt.Sprintf("%s:%s", UUIDPrefix, uuid.New().String())
		}
	}
}

func sanityCheckForAdding(ctx context.Context, d *lab.ChromeOSDevice, q *datastore.Query) error {
	id := d.GetId().GetValue()
	hostname := utils.GetHostname(d)
	var devs []*DeviceEntity
	if err := datastore.GetAll(ctx, q.Eq("Hostname", hostname), &devs); err != nil {
		return errors.Annotate(err, "failed to get host by hostname %s", hostname).Err()
	}
	if len(devs) > 0 {
		return errors.Reason("fail to add device <%s:%s> due to hostname confliction", hostname, id).Err()
	}
	newEntity := DeviceEntity{
		ID:     DeviceEntityID(id),
		Parent: fakeAcestorKey(ctx),
	}
	if !strings.HasPrefix(id, UUIDPrefix) {
		if err := datastore.Get(ctx, &newEntity); err != datastore.ErrNoSuchEntity {
			return errors.Reason("failed to add device %s due to ID conflication", newEntity).Err()
		}
	}
	return nil
}

// AddDevices creates a new Device datastore entity with a unique ID.
func AddDevices(ctx context.Context, devices []*lab.ChromeOSDevice) (*DeviceOpResults, error) {
	updatedTime := time.Now().UTC()

	addMissingID(devices)

	addingResults := make(DeviceOpResults, len(devices))
	for i, d := range devices {
		addingResults[i].Device = d
	}

	f := func(ctx context.Context) error {
		q := datastore.NewQuery(DeviceKind).Ancestor(fakeAcestorKey(ctx))
		entities := make([]*DeviceEntity, 0, len(devices))
		entityResults := make([]*DeviceOpResult, 0, len(devices))
		// Don't use the value returned by `range`. It's a copied value,
		// instead of a reference.
		for i := range addingResults {
			devToAdd := &addingResults[i]
			if err := sanityCheckForAdding(ctx, devToAdd.Device, q); err != nil {
				devToAdd.logError(err)
				continue
			}
			hostname := utils.GetHostname(devToAdd.Device)
			id := devToAdd.Device.GetId().GetValue()

			labConfig, err := proto.Marshal(devToAdd.Device)
			if err != nil {
				devToAdd.logError(errors.Annotate(err, fmt.Sprintf("fail to marshal device <%s:%s>", hostname, id), err).Err())
				continue
			}
			devToAdd.Entity = DeviceEntity{
				ID:        DeviceEntityID(id),
				Hostname:  hostname,
				Updated:   updatedTime,
				LabConfig: labConfig,
				Parent:    fakeAcestorKey(ctx),
			}

			entities = append(entities, &(devToAdd.Entity))
			entityResults = append(entityResults, devToAdd)
		}
		if err := datastore.Put(ctx, entities); err != nil {
			for i, e := range err.(errors.MultiError) {
				if e == nil {
					continue
				}
				entityResults[i].logError(e)
			}
		}
		return nil
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return &addingResults, err
	}
	return &addingResults, nil
}
