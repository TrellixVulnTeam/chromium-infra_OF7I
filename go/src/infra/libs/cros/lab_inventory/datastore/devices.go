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
		if d.GetId() == nil || d.GetId().GetValue() == "" {
			d.Id = &lab.ChromeOSDeviceID{
				Value: fmt.Sprintf("%s:%s", UUIDPrefix, uuid.New().String()),
			}
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
			hostname := utils.GetHostname(devToAdd.Device)
			id := devToAdd.Device.GetId().GetValue()

			devToAdd.Entity = &DeviceEntity{
				ID:       DeviceEntityID(id),
				Hostname: hostname,
				Parent:   fakeAcestorKey(ctx),
			}

			if err := sanityCheckForAdding(ctx, devToAdd.Device, q); err != nil {
				devToAdd.logError(err)
				continue
			}

			labConfig, err := proto.Marshal(devToAdd.Device)
			if err != nil {
				devToAdd.logError(errors.Annotate(err, fmt.Sprintf("fail to marshal device <%s:%s>", hostname, id), err).Err())
				continue
			}
			devToAdd.Entity.Updated = updatedTime
			devToAdd.Entity.LabConfig = labConfig

			entities = append(entities, devToAdd.Entity)
			fmt.Println("add dev", devToAdd.Entity)
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

// DeleteDevicesByIds deletes entities by specified Ids.
// The datastore implementation doesn't raise error when deleting non-existing
// entities: https://github.com/googleapis/google-cloud-go/issues/501
func DeleteDevicesByIds(ctx context.Context, ids []string) DeviceOpResults {
	removingResults := make(DeviceOpResults, len(ids))
	entities := make([]*DeviceEntity, len(ids))
	for i, id := range ids {
		entities[i] = &DeviceEntity{
			ID:     DeviceEntityID(id),
			Parent: fakeAcestorKey(ctx),
		}
		removingResults[i].Entity = entities[i]
	}
	if err := datastore.Delete(ctx, entities); err != nil {
		for i, e := range err.(errors.MultiError) {
			if e == nil {
				continue
			}
			removingResults[i].logError(e)
		}
	}
	return removingResults
}

// DeleteDevicesByHostnames deletes entities by specified hostnames.
func DeleteDevicesByHostnames(ctx context.Context, hostnames []string) DeviceOpResults {
	q := datastore.NewQuery(DeviceKind).Ancestor(fakeAcestorKey(ctx))
	removingResults := make(DeviceOpResults, len(hostnames))
	entities := make([]*DeviceEntity, 0, len(hostnames))
	entityResults := make([]*DeviceOpResult, 0, len(hostnames))

	// Filter out invalid input hostnames.
	for i, hostname := range hostnames {
		removingResults[i].Entity = &DeviceEntity{Hostname: hostname}
		var devs []*DeviceEntity
		if err := datastore.GetAll(ctx, q.Eq("Hostname", hostname), &devs); err != nil {
			removingResults[i].logError(errors.Annotate(err, "failed to get host by hostname %s", hostname).Err())
			continue
		}
		if len(devs) == 0 {
			// Don't raise any error when there's no entities match
			// the hostname. This is consistent with the behavior of
			// removing by ID.
			continue
		}
		if len(devs) > 1 {
			removingResults[i].logError(errors.Reason("multiple entities found with hostname %s: %v", hostname, devs).Err())
			continue
		}
		removingResults[i].Entity = devs[0]
		entities = append(entities, devs[0])
		entityResults = append(entityResults, &removingResults[i])
	}
	if err := datastore.Delete(ctx, entities); err != nil {
		for i, e := range err.(errors.MultiError) {
			if e == nil {
				continue
			}
			entityResults[i].logError(e)
		}
	}
	return removingResults
}

// TODO (guocb) Get HWID data and device config data.

// GetDevicesByIds returns entities by specified ids.
func GetDevicesByIds(ctx context.Context, ids []string) *DeviceOpResults {
	retrievingResults := make(DeviceOpResults, len(ids))
	entities := make([]DeviceEntity, len(ids))
	for i, id := range ids {
		retrievingResults[i].Entity = &entities[i]
		entities[i].ID = DeviceEntityID(id)
		entities[i].Parent = fakeAcestorKey(ctx)
	}
	if err := datastore.Get(ctx, entities); err != nil {
		for i, e := range err.(errors.MultiError) {
			if e == nil {
				continue
			}
			retrievingResults[i].logError(e)
		}
	}
	return &retrievingResults
}

// GetDevicesByHostnames returns entities by specified hostnames.
func GetDevicesByHostnames(ctx context.Context, hostnames []string) *DeviceOpResults {
	q := datastore.NewQuery(DeviceKind).Ancestor(fakeAcestorKey(ctx))
	retrievingResults := make(DeviceOpResults, len(hostnames))

	// Filter out invalid input hostnames.
	for i, hostname := range hostnames {
		var devs []*DeviceEntity
		if err := datastore.GetAll(ctx, q.Eq("Hostname", hostname), &devs); err != nil {
			retrievingResults[i].logError(errors.Annotate(err, "failed to get host by hostname %s", hostname).Err())
			continue
		}
		if len(devs) == 0 {
			retrievingResults[i].logError(errors.Reason("No such host: %s", hostname).Err())
			continue
		}
		if len(devs) > 1 {
			retrievingResults[i].logError(errors.Reason("multiple hosts found with hostname %s: %v", hostname, devs).Err())
			continue
		}
		retrievingResults[i].Entity = devs[0]
	}
	return &retrievingResults
}

// GetAllDevices returns all device entities.
func GetAllDevices(ctx context.Context) ([]*DeviceEntity, error) {
	q := datastore.NewQuery(DeviceKind).Ancestor(fakeAcestorKey(ctx))
	var devs []*DeviceEntity
	if err := datastore.GetAll(ctx, q, &devs); err != nil {
		return nil, errors.Annotate(err, "failed to get all hosts").Err()
	}
	return devs, nil
}
