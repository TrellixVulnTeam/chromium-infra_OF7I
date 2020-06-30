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
	"go.chromium.org/chromiumos/infra/proto/go/manufacturing"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/libs/cros/lab_inventory/changehistory"
	"infra/libs/cros/lab_inventory/utilization"
	"infra/libs/cros/lab_inventory/utils"
)

const (
	// UUIDPrefix is the prefix we used to identify the system generated ID.
	UUIDPrefix       = "UUID"
	dutIDPlaceholder = "IGNORED"
)

// A query in transaction requires to have Ancestor filter, see
// https://cloud.google.com/appengine/docs/standard/python/datastore/query-restrictions#queries_inside_transactions_must_include_ancestor_filters
func fakeAcestorKey(ctx context.Context) *datastore.Key {
	return datastore.MakeKey(ctx, DeviceKind, "key")
}

func addMissingID(devices []*lab.ChromeOSDevice) {
	// Use uuid as the device ID if asset id is not present.
	for _, d := range devices {
		// TODO (guocb) Erase the id passed in as long as it's not asset id to
		// ensure the ID is unique.
		if d.GetId() == nil || d.GetId().GetValue() == "" || d.GetId().GetValue() == dutIDPlaceholder {
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
			return errors.Reason("failed to add device %s due to ID confliction", newEntity).Err()
		}
	}
	return nil
}

func getDutServo(ctx context.Context, d *lab.ChromeOSDevice) (*lab.Servo, error) {
	id := d.GetId().GetValue()
	entity := &DeviceEntity{
		ID:     DeviceEntityID(id),
		Parent: fakeAcestorKey(ctx),
	}
	if err := datastore.Get(ctx, entity); err != nil {
		if datastore.IsErrNoSuchEntity(err) {
			// hiding error, device not exist and cannot provide old servo
			logging.Errorf(ctx, "device with ID: %s not exits", id)
			return nil, nil
		}
		logging.Errorf(ctx, "Failed to get entity from datastore: %s", err)
		return nil, errors.Annotate(err, "Internal error when try to find the device with id: %s", id).Err()
	}
	var labConfig lab.ChromeOSDevice
	if err := entity.GetCrosDeviceProto(&labConfig); err != nil {
		return nil, errors.Annotate(err, "failed to unmarshal lab config data for %s", id).Err()
	}
	dut := labConfig.GetDut()
	if dut == nil {
		return nil, nil
	}
	return dut.GetPeripherals().GetServo(), nil
}

// AddDevices creates a new Device datastore entity with a unique ID.
func AddDevices(ctx context.Context, devices []*lab.ChromeOSDevice, assignServoPort bool) (*DeviceOpResults, error) {
	updatedTime := time.Now().UTC()

	addMissingID(devices)

	addingResults := make(DeviceOpResults, len(devices))
	for i, d := range devices {
		addingResults[i].Data = d
	}

	r := newServoHostRegistryFromProtoMsgs(ctx, devices)

	f := func(ctx context.Context) error {
		q := datastore.NewQuery(DeviceKind).Ancestor(fakeAcestorKey(ctx))
		entities := make([]*DeviceEntity, 0, len(devices))
		entityResults := make([]*DeviceOpResult, 0, len(devices))
		// Don't use the value returned by `range`. It's a copied value,
		// instead of a reference.
		for i := range addingResults {
			devToAdd := &addingResults[i]
			message := devToAdd.Data.(*lab.ChromeOSDevice)
			hostname := utils.GetHostname(message)
			id := message.GetId().GetValue()

			devToAdd.Entity = &DeviceEntity{
				ID:       DeviceEntityID(id),
				Hostname: hostname,
				Parent:   fakeAcestorKey(ctx),
			}

			if err := sanityCheckForAdding(ctx, message, q); err != nil {
				devToAdd.logError(err)
				continue
			}

			if dut := message.GetDut(); dut != nil {
				// Update associated labstation if the DUT has a new servo. Also
				// assign new servo port if specified.
				if err := r.amendServoToLabstation(ctx, dut, nil, assignServoPort); err != nil {
					devToAdd.logError(err)
					continue
				}
			}

			labConfig, err := proto.Marshal(message)
			if err != nil {
				devToAdd.logError(errors.Annotate(err, fmt.Sprintf("fail to marshal device <%s:%s>", hostname, id), err).Err())
				continue
			}
			devToAdd.Entity.Updated = updatedTime
			devToAdd.Entity.LabConfig = labConfig

			entities = append(entities, devToAdd.Entity)
			entityResults = append(entityResults, devToAdd)
		}
		if err := datastore.Put(ctx, entities); err != nil {
			for i, e := range err.(errors.MultiError) {
				entityResults[i].logError(e)
			}
		}
		logLifeCycleEvent(ctx, changehistory.LifeCycleDeployment, addingResults)
		return r.saveToDatastore(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return &addingResults, err
	}
	return &addingResults, nil
}

// DeleteDevicesByIds deletes entities by specified Ids.
// The datastore implementation doesn't raise error when deleting non-existing
// entities: https://github.com/googleapis/google-cloud-go/issues/501
//
// As additional deleting servo from labstation for deleted device.
func DeleteDevicesByIds(ctx context.Context, ids []string) DeviceOpResults {
	removingResults := make(DeviceOpResults, len(ids))
	r := newServoHostRegistryFromProtoMsgs(ctx, nil)

	f := func(ctx context.Context) error {
		removingResults = GetDevicesByIds(ctx, ids)
		var entities []*DeviceEntity
		for i := range removingResults {
			deviceResult := &removingResults[i]
			if deviceResult.Err != nil || deviceResult.Entity == nil {
				continue
			}
			entity := deviceResult.Entity
			var devProto lab.ChromeOSDevice
			if err := entity.GetCrosDeviceProto(&devProto); err != nil {
				deviceResult.logError(errors.Annotate(err, "failed to unmarshal lab config data for %s", entity.Hostname).Err())
				continue
			}
			if dut := devProto.GetDut(); dut != nil {
				err := r.removeDeviceFromLabstation(ctx, dut)
				if err != nil {
					deviceResult.logError(errors.Annotate(err, "failed to delete servo from labstation for: %s", entity.Hostname).Err())
					continue
				}
			} else if labstation := devProto.GetLabstation(); labstation != nil {
				if len(labstation.GetServos()) > 0 {
					deviceResult.logError(errors.Reason("cannot delete labstation: %s used by the DUTs", entity.Hostname).Err())
					continue
				}
			}
			entities = append(entities, entity)
		}
		if err := datastore.Delete(ctx, entities); err != nil {
			for i, e := range err.(errors.MultiError) {
				removingResults[i].logError(e)
			}
		}
		logLifeCycleEvent(ctx, changehistory.LifeCycleDecomm, removingResults)
		return r.saveToDatastore(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		for _, result := range removingResults {
			result.logError(err)
		}
	}
	return removingResults
}

func logLifeCycleEvent(ctx context.Context, event changehistory.LifeCycleEvent, opResults DeviceOpResults) {
	var changes changehistory.Changes
	for _, r := range opResults.Passed() {
		switch event {
		case changehistory.LifeCycleDeployment:
			changes.LogDeployment(string(r.Entity.ID), r.Entity.Hostname)
		case changehistory.LifeCycleDecomm:
			changes.LogDecommission(string(r.Entity.ID), r.Entity.Hostname)
		}
		logging.Infof(ctx, "LifeCycleEvent: %s %s", event, r.Entity)
	}
	if len(changes) == 0 {
		return
	}
	if err := changes.SaveToDatastore(ctx); err != nil {
		logging.Errorf(ctx, "%s: Failed to save change history to datastore: %s", event, err)
	}
}

// DeleteDevicesByHostnames deletes entities by specified hostnames.
//
// As additional deleting servo from labstation for deleted device.
func DeleteDevicesByHostnames(ctx context.Context, hostnames []string) DeviceOpResults {
	removingResults := make(DeviceOpResults, len(hostnames))
	r := newServoHostRegistryFromProtoMsgs(ctx, nil)

	f := func(ctx context.Context) error {
		q := datastore.NewQuery(DeviceKind).Ancestor(fakeAcestorKey(ctx))
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
				removingResults[i].logError(errors.Reason("No such host: %s", hostname).Err())
				continue
			}
			if len(devs) > 1 {
				removingResults[i].logError(errors.Reason("multiple entities found with hostname %s: %v", hostname, devs).Err())
				continue
			}
			entity := devs[0]
			var devProto lab.ChromeOSDevice
			if err := entity.GetCrosDeviceProto(&devProto); err != nil {
				removingResults[i].logError(errors.Annotate(err, "failed to unmarshal lab config data for %s", hostname).Err())
				continue
			}
			if dut := devProto.GetDut(); dut != nil {
				err := r.removeDeviceFromLabstation(ctx, dut)
				if err != nil {
					removingResults[i].logError(errors.Annotate(err, "failed to delete servo from labstation for: %s", hostname).Err())
					continue
				}
			} else if labstation := devProto.GetLabstation(); labstation != nil {
				if len(labstation.GetServos()) > 0 {
					removingResults[i].logError(errors.Reason("cannot delete labstation: %s used by the DUTs", hostname).Err())
					continue
				}
			}

			removingResults[i].Entity = entity
			entities = append(entities, entity)
			entityResults = append(entityResults, &removingResults[i])
		}
		if err := datastore.Delete(ctx, entities); err != nil {
			for i, e := range err.(errors.MultiError) {
				entityResults[i].logError(e)
			}
		}
		logLifeCycleEvent(ctx, changehistory.LifeCycleDecomm, removingResults)
		return r.saveToDatastore(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		for _, result := range removingResults {
			result.logError(err)
		}
	}
	return removingResults
}

// GetDevicesByIds returns entities by specified ids.
func GetDevicesByIds(ctx context.Context, ids []string) DeviceOpResults {
	retrievingResults := make(DeviceOpResults, len(ids))
	entities := make([]DeviceEntity, len(ids))
	for i, id := range ids {
		retrievingResults[i].Entity = &entities[i]
		entities[i].ID = DeviceEntityID(id)
		entities[i].Parent = fakeAcestorKey(ctx)
	}
	if err := datastore.Get(ctx, entities); err != nil {
		for i, e := range err.(errors.MultiError) {
			retrievingResults[i].logError(e)
		}
	}
	return retrievingResults
}

// GetDevicesByHostnames returns entities by specified hostnames.
func GetDevicesByHostnames(ctx context.Context, hostnames []string) DeviceOpResults {
	q := datastore.NewQuery(DeviceKind).Ancestor(fakeAcestorKey(ctx))
	retrievingResults := make(DeviceOpResults, len(hostnames))

	// Filter out invalid input hostnames.
	for i, hostname := range hostnames {
		retrievingResults[i].Entity = &DeviceEntity{Hostname: hostname}
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
	return retrievingResults
}

// GetAllDevices  returns all device entities.
//
// TODO(guocb) optimize for performance if needed.
func GetAllDevices(ctx context.Context) (DeviceOpResults, error) {
	q := datastore.NewQuery(DeviceKind).Ancestor(fakeAcestorKey(ctx))
	var devs []*DeviceEntity
	if err := datastore.GetAll(ctx, q, &devs); err != nil {
		return nil, errors.Annotate(err, "failed to get all devices").Err()
	}
	result := make([]DeviceOpResult, len(devs))
	for i, d := range devs {
		result[i].Entity = d
	}
	return DeviceOpResults(result), nil
}

// GetDevicesByModels returns all device entities of models.
//
// TODO(guocb) optimize for performance if needed.
func GetDevicesByModels(ctx context.Context, models []string) (DeviceOpResults, error) {
	if len(models) == 0 {
		return nil, nil
	}
	q := datastore.NewQuery(DeviceKind).Ancestor(fakeAcestorKey(ctx))
	var devs []*DeviceEntity
	if err := datastore.GetAll(ctx, q, &devs); err != nil {
		return nil, errors.Annotate(err, "failed to get all devices").Err()
	}
	modelSet := stringset.NewFromSlice(models...)
	result := make([]DeviceOpResult, 0, len(devs))
	for _, d := range devs {
		var labConfig lab.ChromeOSDevice
		if err := d.GetCrosDeviceProto(&labConfig); err != nil {
			logging.Errorf(ctx, "failed to unmarshal lab config data for %s", d.Hostname)
			continue
		}
		if !modelSet.Has(labConfig.GetDeviceConfigId().GetModelId().GetValue()) {
			continue
		}

		result = append(result, DeviceOpResult{Entity: d})
	}
	return DeviceOpResults(result), nil
}

// UpdateDeviceID of the old device to the new device
//
// Changes the timestamp to reflect this change.
func UpdateDeviceID(ctx context.Context, oldDev, newDev string) error {
	if oldDev == "" || newDev == "" {
		return errors.Reason("UpdateDeviceID, Invalid input").Err()
	}
	updatedTime := time.Now().UTC()

	oldEntity := &DeviceEntity{
		ID:     DeviceEntityID(oldDev),
		Parent: fakeAcestorKey(ctx),
	}

	newEntity := &DeviceEntity{
		ID:      DeviceEntityID(newDev),
		Parent:  fakeAcestorKey(ctx),
		Updated: updatedTime,
	}

	f := func(ctx context.Context) error {
		if err := datastore.Get(ctx, oldEntity); err != nil {
			return err
		}
		// Generate lab config
		var labConfig lab.ChromeOSDevice
		err := proto.Unmarshal(oldEntity.LabConfig, &labConfig)
		if err != nil {
			return err
		}
		labConfig.Id = &lab.ChromeOSDeviceID{Value: newDev}
		l, err := proto.Marshal(&labConfig)
		newEntity.LabConfig = l

		// Update Dut State
		var state lab.DutState
		if err := oldEntity.GetDutStateProto(&state); err != nil {
			return err
		}
		state.Id = &lab.ChromeOSDeviceID{Value: newDev}
		mState, err := proto.Marshal(&state)
		if err != nil {
			return err
		}
		newEntity.DutState = mState
		newEntity.Hostname = oldEntity.Hostname

		if err := datastore.Delete(ctx, oldEntity); err != nil {
			return err
		}
		if err := datastore.Put(ctx, newEntity); err != nil {
			return err
		}
		return nil
	}
	return datastore.RunInTransaction(ctx, f, nil)
}

func updateEntities(ctx context.Context, opResults DeviceOpResults, additionalFilter func()) func(context.Context) error {
	maxLen := len(opResults)
	entities := make([]*DeviceEntity, maxLen)
	for i := range opResults {
		entities[i] = opResults[i].Entity
	}
	f := func(ctx context.Context) error {
		if err := datastore.Get(ctx, entities); err != nil {
			for i, e := range err.(errors.MultiError) {
				opResults[i].logError(errors.Annotate(e, "failed to get entities").Err())
			}
		}
		if additionalFilter != nil {
			additionalFilter()
		}
		entities = []*DeviceEntity{}
		entityIndexes := make([]int, 0, maxLen)
		updatedTime := time.Now().UTC()
		var changes changehistory.Changes
		for i, r := range opResults {
			if r.Err != nil {
				continue
			}
			c, err := r.Entity.UpdatePayload(r.Data, updatedTime)
			if err != nil {
				r.logError(errors.Annotate(err, "failed to update payload").Err())
				continue
			}
			changes = append(changes, c...)
			entities = append(entities, r.Entity)
			entityIndexes = append(entityIndexes, i)
		}
		if err := changes.SaveToDatastore(ctx); err != nil {
			logging.Errorf(ctx, "UpdateEntities: Failed to save change history to datastore: %s", err)
		}
		if err := datastore.Put(ctx, entities); err != nil {
			merr, ok := err.(errors.MultiError)
			if !ok {
				return err
			}
			for i, e := range merr {
				opResults[entityIndexes[i]].logError(errors.Annotate(e, "failed to save entity to datastore").Err())
			}
		}
		return nil
	}
	return f
}

// UpdateDeviceSetup updates the content of lab.ChromeOSDevice.
func UpdateDeviceSetup(ctx context.Context, devices []*lab.ChromeOSDevice, assignServoPort bool) (DeviceOpResults, error) {
	updatingResults := make(DeviceOpResults, len(devices))
	entities := make([]*DeviceEntity, len(devices))
	r := newServoHostRegistryFromProtoMsgs(ctx, devices)
	for i, d := range devices {
		entities[i] = &DeviceEntity{
			ID:     DeviceEntityID(d.GetId().GetValue()),
			Parent: fakeAcestorKey(ctx),
		}
		updatingResults[i].Data = devices[i]
		updatingResults[i].Entity = entities[i]

		if dut := devices[i].GetDut(); dut != nil {
			oldServo, err := getDutServo(ctx, d)
			if err != nil {
				return nil, err
			}
			if err := r.amendServoToLabstation(ctx, dut, oldServo, assignServoPort); err != nil {
				return nil, err
			}
		}
	}
	f := func(ctx context.Context) error {
		err := updateEntities(ctx, updatingResults, nil)(ctx)
		if err != nil {
			return errors.Annotate(err, "save updated entities").Err()
		}
		if err := r.saveToDatastore(ctx); err != nil {
			return errors.Annotate(err, "save changed labstations caused by updated DUT").Err()
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return updatingResults, err
	}
	return updatingResults, nil
}

// DutMeta refers to the metadata to be stored for a DUT.
type DutMeta struct {
	SerialNumber string
	HwID         string
}

// UpdateDutMeta updates dut serial number and hwid for a given host.
func UpdateDutMeta(ctx context.Context, meta map[string]DutMeta) (DeviceOpResults, error) {
	ids := make([]string, 0, len(meta))
	for i := range meta {
		ids = append(ids, i)
	}
	results := GetDevicesByIds(ctx, ids)

	var updateResults DeviceOpResults
	var failedResults DeviceOpResults
	for _, r := range results {
		if r.Err != nil {
			failedResults = append(failedResults, r)
			continue
		}
		var labData lab.ChromeOSDevice
		if err := r.Entity.GetCrosDeviceProto(&labData); err != nil {
			r.logError(err)
			logging.Debugf(ctx, "fail to parse proto for entity: %#v", r.Entity)
			failedResults = append(failedResults, r)
			continue
		}
		hid := string(r.Entity.ID)
		if labData.SerialNumber == meta[hid].SerialNumber && labData.ManufacturingId != nil && labData.GetManufacturingId().GetValue() == meta[hid].HwID {
			r.logError(errors.New(fmt.Sprintf("meta is not changed. Old serial number %s, old hwid %s", meta[hid].SerialNumber, meta[hid].HwID)))
			failedResults = append(failedResults, r)
			continue
		}
		labData.SerialNumber = meta[hid].SerialNumber
		if labData.ManufacturingId == nil {
			labData.ManufacturingId = &manufacturing.ConfigID{
				Value: meta[hid].HwID,
			}
		} else {
			labData.ManufacturingId.Value = meta[hid].HwID
		}

		r := DeviceOpResult{
			Entity: &DeviceEntity{
				ID:     r.Entity.ID,
				Parent: fakeAcestorKey(ctx),
			},
			Data: &labData,
		}
		updateResults = append(updateResults, r)
	}
	f := updateEntities(ctx, updateResults, nil)

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return updateResults, err
	}
	return append(updateResults, failedResults...), nil
}

// LabMeta refers to the metadata to be stored for a DUT.
type LabMeta struct {
	ServoType   string
	SmartUsbhub bool
}

// UpdateLabMeta updates servo_type and smart_usbhub flag for a given host.
func UpdateLabMeta(ctx context.Context, meta map[string]LabMeta) (DeviceOpResults, error) {
	ids := make([]string, 0, len(meta))
	for i := range meta {
		ids = append(ids, i)
	}
	results := GetDevicesByIds(ctx, ids)

	var updateResults DeviceOpResults
	var failedResults DeviceOpResults
	for _, r := range results {
		if r.Err != nil {
			failedResults = append(failedResults, r)
			continue
		}
		var labData lab.ChromeOSDevice
		if err := r.Entity.GetCrosDeviceProto(&labData); err != nil {
			r.logError(err)
			logging.Debugf(ctx, "fail to parse proto for entity: %#v", r.Entity)
			failedResults = append(failedResults, r)
			continue
		}

		hid := string(r.Entity.ID)
		if dut := labData.GetDut(); dut != nil {
			p := dut.GetPeripherals()
			if servo := p.GetServo(); servo != nil {
				servo.ServoType = meta[hid].ServoType
			}
			p.SmartUsbhub = meta[hid].SmartUsbhub
		}

		r := DeviceOpResult{
			Entity: &DeviceEntity{
				ID:     r.Entity.ID,
				Parent: fakeAcestorKey(ctx),
			},
			Data: &labData,
		}
		updateResults = append(updateResults, r)
	}
	f := updateEntities(ctx, updateResults, nil)

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return updateResults, err
	}
	return append(updateResults, failedResults...), nil
}

// UpdateDutsStatus updates dut status of testing related.
func UpdateDutsStatus(ctx context.Context, states []*lab.DutState) (DeviceOpResults, error) {
	maxLen := len(states)
	updatingResults := make(DeviceOpResults, maxLen)
	entities := make([]*DeviceEntity, maxLen)
	// The Id must be a valid Id of DeviceUnderTest.
	for i, s := range states {
		entities[i] = &DeviceEntity{
			ID:     DeviceEntityID(s.GetId().GetValue()),
			Parent: fakeAcestorKey(ctx),
		}
		updatingResults[i].Data = states[i]
		updatingResults[i].Entity = entities[i]
	}
	filter := func() {
		// The returned device must be DeviceUnderTest.
		var d lab.ChromeOSDevice
		for i, e := range entities {
			if err := e.GetCrosDeviceProto(&d); err != nil {
				updatingResults[i].logError(errors.Annotate(err, "failed to get proto of entity %v", e).Err())
				continue
			}
			if d.GetDut() == nil {
				updatingResults[i].logError(errors.Reason("entity %v isn't a DUT", e).Err())
				continue
			}
		}
	}
	// We cannot filter entities inside `updateEntities` as the filtering is
	// after on the entity retrieving.
	f := updateEntities(ctx, updatingResults, filter)

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return updatingResults, err
	}
	return updatingResults, nil
}

// DeviceProperty specifies some device property.
type DeviceProperty struct {
	Hostname        string
	Pool            string
	PowerunitName   string
	PowerunitOutlet string
}

// BatchUpdateDevices updates devices of some specific properties in a batch.
func BatchUpdateDevices(ctx context.Context, duts []*DeviceProperty) error {
	var hostnames []string
	propertyMap := map[string]*DeviceProperty{}
	for _, d := range duts {
		hostnames = append(hostnames, d.Hostname)
		propertyMap[d.Hostname] = d
	}
	now := time.Now().UTC()
	setRpm := func(rpm *lab.RPM, p *DeviceProperty) {
		if p.PowerunitName != "" {
			rpm.PowerunitName = p.PowerunitName
		}
		if p.PowerunitOutlet != "" {
			rpm.PowerunitOutlet = p.PowerunitOutlet
		}
	}
	f := func(ctx context.Context) error {
		var changes changehistory.Changes
		entities := make([]*DeviceEntity, 0, len(duts))
		for _, r := range GetDevicesByHostnames(ctx, hostnames).Passed() {
			var labConfig lab.ChromeOSDevice
			if err := r.Entity.GetCrosDeviceProto(&labConfig); err != nil {
				logging.Errorf(ctx, "Cannot get lab config from entity %v: %s", r.Entity, err.Error())
				continue
			}
			p := propertyMap[r.Entity.Hostname]
			if dut := labConfig.GetDut(); dut != nil {
				if p.Pool != "" {
					dut.Pools = []string{p.Pool}
				}
				if peri := dut.GetPeripherals(); peri == nil {
					dut.Peripherals = &lab.Peripherals{
						Rpm: &lab.RPM{},
					}
				} else if peri.GetRpm() == nil {
					peri.Rpm = &lab.RPM{}
				}
				setRpm(dut.GetPeripherals().GetRpm(), p)
			}
			if labstation := labConfig.GetLabstation(); labstation != nil {
				if p.Pool != "" {
					labstation.Pools = []string{p.Pool}
				}
				if labstation.GetRpm() == nil {
					labstation.Rpm = &lab.RPM{}
				}
				setRpm(labstation.GetRpm(), p)
			}
			c, err := r.Entity.UpdatePayload(&labConfig, now)
			if err != nil {
				r.logError(errors.Annotate(err, "failed to update payload").Err())
				continue
			}
			changes = append(changes, c...)
			entities = append(entities, r.Entity)
		}
		if err := changes.SaveToDatastore(ctx); err != nil {
			logging.Errorf(ctx, "BatchUpdateDevices: Failed to save change history to datastore: %s", err)
		}
		return datastore.Put(ctx, entities)
	}
	return datastore.RunInTransaction(ctx, f, nil)
}

// ReportInventory reports the inventory metrics.
func ReportInventory(ctx context.Context, environment string) error {
	results, err := GetAllDevices(ctx)
	if err != nil {
		return errors.Annotate(err, "report inventory").Err()
	}
	var devices []*lab.ChromeOSDevice
	for _, r := range results {
		var d lab.ChromeOSDevice
		if err := r.Entity.GetCrosDeviceProto(&d); err != nil {
			logging.Errorf(ctx, "failed to get device proto: %v", r.Entity)
			continue
		}
		devices = append(devices, &d)
	}
	utilization.ReportInventoryMetricsV2(ctx, devices, environment)
	return nil
}
