// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"fmt"
	"strings"

	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/chromiumos/infra/proto/go/manufacturing"
	ds "go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/grpcutil"
	"golang.org/x/net/context"

	api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/appengine/cros/lab_inventory/app/config"
	"infra/libs/cros/lab_inventory/changehistory"
	"infra/libs/cros/lab_inventory/datastore"
	"infra/libs/cros/lab_inventory/deviceconfig"
	"infra/libs/cros/lab_inventory/dronecfg"
	"infra/libs/cros/lab_inventory/hwid"
	"infra/libs/cros/lab_inventory/manufacturingconfig"
	"infra/libs/cros/lab_inventory/utils"
)

// InventoryServerImpl implements service interfaces.
type InventoryServerImpl struct {
}

var (
	getHwidDataFunc            = hwid.GetHwidData
	getDeviceConfigFunc        = deviceconfig.GetCachedConfig
	getManufacturingConfigFunc = manufacturingconfig.GetCachedConfig
)

func getPassedResults(results []datastore.DeviceOpResult) []*api.DeviceOpResult {
	passedDevices := make([]*api.DeviceOpResult, 0, len(results))
	for _, res := range datastore.DeviceOpResults(results).Passed() {
		r := new(api.DeviceOpResult)
		r.Id = string(res.Entity.ID)
		r.Hostname = res.Entity.Hostname
		passedDevices = append(passedDevices, r)
	}
	return passedDevices
}

func getFailedResults(results []datastore.DeviceOpResult, hideUUID bool) []*api.DeviceOpResult {
	failedDevices := make([]*api.DeviceOpResult, 0, len(results))
	for _, res := range datastore.DeviceOpResults(results).Failed() {
		r := new(api.DeviceOpResult)
		r.Hostname = res.Entity.Hostname
		r.ErrorMsg = res.Err.Error()
		id := string(res.Entity.ID)
		if !(hideUUID && strings.HasPrefix(id, datastore.UUIDPrefix)) {
			r.Id = id
		}
		failedDevices = append(failedDevices, r)
	}
	return failedDevices
}

func updateDroneCfg(ctx context.Context, devices []*api.DeviceOpResult, addDuts bool) (err error) {
	// Merge the new DUTs to drones.
	var duts []dronecfg.DUT
	for _, d := range devices {
		duts = append(duts, dronecfg.DUT{Hostname: d.Hostname, ID: d.Id})
	}
	toChange := []dronecfg.Entity{
		{
			Hostname: dronecfg.QueenDroneName(config.Get(ctx).Environment),
			DUTs:     duts,
		},
	}
	if addDuts {
		err = dronecfg.MergeDutsToDrones(ctx, toChange, nil)
	} else {
		err = dronecfg.MergeDutsToDrones(ctx, nil, toChange)
	}
	if err != nil {
		err = errors.Annotate(err, "update drone config").Err()
	}
	return err
}

// AddCrosDevices adds new Chrome OS devices to the inventory.
func (is *InventoryServerImpl) AddCrosDevices(ctx context.Context, req *api.AddCrosDevicesRequest) (resp *api.AddCrosDevicesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err = req.Validate(); err != nil {
		return nil, err
	}
	addingResults, err := datastore.AddDevices(ctx, req.Devices, req.PickServoPort)
	if err != nil {
		return nil, errors.Annotate(err, "internal error").Tag(grpcutil.InternalTag).Err()
	}
	passedDevices := getPassedResults(*addingResults)
	if err := updateDroneCfg(ctx, passedDevices, true); err != nil {
		return nil, errors.Annotate(err, "add cros devices").Err()
	}

	failedDevices := getFailedResults(*addingResults, true)
	resp = &api.AddCrosDevicesResponse{
		PassedDevices: passedDevices,
		FailedDevices: failedDevices,
	}
	return resp, nil
}

func addFailedDevice(ctx context.Context, failedDevices *[]*api.DeviceOpResult, dev *lab.ChromeOSDevice, err error, operation string) {
	hostname := utils.GetHostname(dev)
	logging.Errorf(ctx, "failed to %s for %s: %s", operation, hostname, err.Error())
	*failedDevices = append(*failedDevices, &api.DeviceOpResult{
		Id:       dev.GetId().GetValue(),
		Hostname: hostname,
		ErrorMsg: err.Error(),
	})

}

func getDeviceConfigData(ctx context.Context, extendedData []*api.ExtendedDeviceData) ([]*api.ExtendedDeviceData, []*api.DeviceOpResult) {
	// Start to retrieve device config data.
	devCfgIds := make([]*device.ConfigId, len(extendedData))
	for i, d := range extendedData {
		logging.Debugf(ctx, "before convert: %#v", d.LabConfig.DeviceConfigId)
		devCfgIds[i] = deviceconfig.ConvertValidDeviceConfigID(d.LabConfig.DeviceConfigId)
		logging.Debugf(ctx, "real device config ID: %#v", devCfgIds[i])
	}
	devCfgs, err := getDeviceConfigFunc(ctx, devCfgIds)
	newExtendedData := make([]*api.ExtendedDeviceData, 0, len(extendedData))
	failedDevices := make([]*api.DeviceOpResult, 0, len(extendedData))
	for i := range devCfgs {
		if err == nil || err.(errors.MultiError)[i] == nil {
			extendedData[i].DeviceConfig = devCfgs[i].(*device.Config)
			newExtendedData = append(newExtendedData, extendedData[i])
		} else {
			addFailedDevice(ctx, &failedDevices, extendedData[i].LabConfig, err.(errors.MultiError)[i], "get device config data")
		}
	}
	return newExtendedData, failedDevices
}

func getManufacturingConfigData(ctx context.Context, extendedData []*api.ExtendedDeviceData) ([]*api.ExtendedDeviceData, []*api.DeviceOpResult) {
	// Start to retrieve device config data.
	cfgIds := make([]*manufacturing.ConfigID, len(extendedData))
	for i, d := range extendedData {
		cfgIds[i] = d.LabConfig.ManufacturingId
	}
	mCfgs, err := getManufacturingConfigFunc(ctx, cfgIds)
	newExtendedData := make([]*api.ExtendedDeviceData, 0, len(extendedData))
	failedDevices := make([]*api.DeviceOpResult, 0, len(extendedData))
	for i := range mCfgs {
		if err == nil || err.(errors.MultiError)[i] == nil {
			extendedData[i].ManufacturingConfig = mCfgs[i].(*manufacturing.Config)
			newExtendedData = append(newExtendedData, extendedData[i])
		} else {
			// Ignore errors if the ID doesn't exist in manufacturing config.
			if ds.IsErrNoSuchEntity(err.(errors.MultiError)[i]) {
				logging.Errorf(ctx, "No matched manufacturing config found: %s", cfgIds[i])
				newExtendedData = append(newExtendedData, extendedData[i])
				continue
			}

			addFailedDevice(ctx, &failedDevices, extendedData[i].LabConfig, err.(errors.MultiError)[i], "get manufacturing config data")
		}
	}
	return newExtendedData, failedDevices
}

func getExtendedDeviceData(ctx context.Context, devices []datastore.DeviceOpResult) ([]*api.ExtendedDeviceData, []*api.DeviceOpResult) {
	logging.Debugf(ctx, "Get exteneded data for %d devcies", len(devices))
	secret := config.Get(ctx).HwidSecret
	extendedData := make([]*api.ExtendedDeviceData, 0, len(devices))
	failedDevices := make([]*api.DeviceOpResult, 0, len(devices))
	for _, r := range devices {
		var labData lab.ChromeOSDevice
		if err := r.Entity.GetCrosDeviceProto(&labData); err != nil {
			logging.Errorf(ctx, "Wrong lab config data of device entity %s", r.Entity)
			failedDevices = append(failedDevices, &api.DeviceOpResult{
				Id:       string(r.Entity.ID),
				Hostname: r.Entity.Hostname,
				ErrorMsg: err.Error(),
			})
			continue
		}
		var dutState lab.DutState
		if err := r.Entity.GetDutStateProto(&dutState); err != nil {
			addFailedDevice(ctx, &failedDevices, &labData, err, "unmarshal dut state data")
			continue
		}
		hwidData, err := getHwidDataFunc(ctx, labData.GetManufacturingId().GetValue(), secret)
		if err != nil {
			operation := fmt.Sprintf("get HWID data of %s", labData.GetManufacturingId().GetValue())
			addFailedDevice(ctx, &failedDevices, &labData, err, operation)
			continue
		}
		extendedData = append(extendedData, &api.ExtendedDeviceData{
			LabConfig: &labData,
			DutState:  &dutState,
			HwidData: &api.HwidData{
				Sku:     hwidData.Sku,
				Variant: hwidData.Variant,
			},
		})
	}
	// Get device config in a batch.
	extendedData, moreFailedDevices := getDeviceConfigData(ctx, extendedData)
	failedDevices = append(failedDevices, moreFailedDevices...)

	extendedData, moreFailedDevices = getManufacturingConfigData(ctx, extendedData)
	failedDevices = append(failedDevices, moreFailedDevices...)
	return extendedData, failedDevices
}

type requestWithIds interface {
	GetIds() []*api.DeviceID
}

// extractHostnamesAndDeviceIDs extracts hostnames and lab.ChromeOSDeviceIDs
// from the input request.
func extractHostnamesAndDeviceIDs(ctx context.Context, req requestWithIds) ([]string, []string) {
	reqIds := req.GetIds()
	maxLen := len(reqIds)
	hostnames := make([]string, 0, maxLen)
	devIds := make([]string, 0, maxLen)
	for _, id := range reqIds {
		if _, ok := id.GetId().(*api.DeviceID_Hostname); ok {
			hostnames = append(hostnames, id.GetHostname())
		} else {
			devIds = append(devIds, id.GetChromeosDeviceId())
		}
	}
	logging.Debugf(ctx, "There are %d hostnames and %d Chrome OS Device IDs in the request", len(hostnames), len(devIds))
	return hostnames, devIds
}

// GetCrosDevices retrieves requested Chrome OS devices from the inventory.
func (is *InventoryServerImpl) GetCrosDevices(ctx context.Context, req *api.GetCrosDevicesRequest) (resp *api.GetCrosDevicesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()

	if err = req.Validate(); err != nil {
		return nil, err
	}

	hostnames, devIds := extractHostnamesAndDeviceIDs(ctx, req)
	models := req.GetModels()
	result := ([]datastore.DeviceOpResult)(datastore.GetDevicesByIds(ctx, devIds))
	result = append(result, datastore.GetDevicesByHostnames(ctx, hostnames)...)
	byModels, err := datastore.GetDevicesByModels(ctx, models)
	if err != nil {
		return nil, err
	}
	result = append(result, byModels...)

	extendedData, moreFailedDevices := getExtendedDeviceData(ctx, datastore.DeviceOpResults(result).Passed())
	failedDevices := getFailedResults(result, false)
	failedDevices = append(failedDevices, moreFailedDevices...)

	resp = &api.GetCrosDevicesResponse{
		Data:          extendedData,
		FailedDevices: failedDevices,
	}
	return resp, nil
}

func logDeviceOpResults(ctx context.Context, res datastore.DeviceOpResults) {
	for _, r := range res {
		if r.Err == nil {
			logging.Debugf(ctx, "Device ID %s: succeed", r.Entity.ID)
		} else {
			logging.Debugf(ctx, "Device ID %s: %s", r.Entity.ID, r.Err)
		}
	}
}

// UpdateDutsStatus updates selected Duts' status labels related to testing.
func (is *InventoryServerImpl) UpdateDutsStatus(ctx context.Context, req *api.UpdateDutsStatusRequest) (resp *api.UpdateDutsStatusResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()

	if err = req.Validate(); err != nil {
		return nil, err
	}

	meta := make(map[string]datastore.DutMeta, len(req.GetDutMetas()))
	for _, d := range req.GetDutMetas() {
		meta[d.GetChromeosDeviceId()] = datastore.DutMeta{
			SerialNumber: d.GetSerialNumber(),
			HwID:         d.GetHwID(),
		}
	}
	metaUpdateResults, err := datastore.UpdateDutMeta(ctx, meta)
	logging.Debugf(ctx, "Meta update results")
	logDeviceOpResults(ctx, metaUpdateResults)
	if err != nil {
		logging.Errorf(ctx, "fail to update dut meta: %s", err.Error())
		return nil, err
	}

	updatingResults, err := datastore.UpdateDutsStatus(changehistory.Use(ctx, req.Reason), req.States)
	if err != nil {
		return nil, err
	}
	logging.Debugf(ctx, "State update results")
	logDeviceOpResults(ctx, updatingResults)

	updatedDevices := getPassedResults(updatingResults)
	failedDevices := getFailedResults(updatingResults, false)
	resp = &api.UpdateDutsStatusResponse{
		UpdatedDevices: updatedDevices,
		FailedDevices:  failedDevices,
	}
	return resp, nil
}

// UpdateCrosDevicesSetup updates the selected Chrome OS devices setup data in
// the inventory.
func (is *InventoryServerImpl) UpdateCrosDevicesSetup(ctx context.Context, req *api.UpdateCrosDevicesSetupRequest) (resp *api.UpdateCrosDevicesSetupResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()

	if err = req.Validate(); err != nil {
		return nil, err
	}
	updatingResults, err := datastore.UpdateDeviceSetup(changehistory.Use(ctx, req.Reason), req.Devices, req.PickServoPort)
	if err != nil {
		return nil, err
	}

	updatedDevices := getPassedResults(updatingResults)
	// Update dronecfg datastore in case there are any DUTs get renamed.
	if err := updateDroneCfg(ctx, updatedDevices, true); err != nil {
		return nil, errors.Annotate(err, "update cros device setup").Err()
	}

	failedDevices := getFailedResults(updatingResults, false)
	resp = &api.UpdateCrosDevicesSetupResponse{
		UpdatedDevices: updatedDevices,
		FailedDevices:  failedDevices,
	}
	return resp, nil
}

// DeleteCrosDevices delete the selelcted devices from the inventory.
func (is *InventoryServerImpl) DeleteCrosDevices(ctx context.Context, req *api.DeleteCrosDevicesRequest) (resp *api.DeleteCrosDevicesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()

	if err = req.Validate(); err != nil {
		return nil, err
	}
	hostnames, ids := extractHostnamesAndDeviceIDs(ctx, req)
	deletingResults := datastore.DeleteDevicesByIds(ctx, ids)
	deletingResultsByHostname := datastore.DeleteDevicesByHostnames(ctx, hostnames)
	deletingResults = append(deletingResults, deletingResultsByHostname...)

	removedDevices := getPassedResults(deletingResults)
	if err := updateDroneCfg(ctx, removedDevices, false); err != nil {
		return nil, errors.Annotate(err, "delete cros devices").Err()
	}

	failedDevices := getFailedResults(deletingResults, false)
	resp = &api.DeleteCrosDevicesResponse{
		RemovedDevices: removedDevices,
		FailedDevices:  failedDevices,
	}
	return resp, nil
}
