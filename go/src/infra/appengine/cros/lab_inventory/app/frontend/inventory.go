// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"strings"

	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/grpcutil"
	"golang.org/x/net/context"

	api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/appengine/cros/lab_inventory/app/config"
	"infra/libs/cros/lab_inventory/changehistory"
	"infra/libs/cros/lab_inventory/datastore"
	"infra/libs/cros/lab_inventory/deviceconfig"
	"infra/libs/cros/lab_inventory/hwid"
	"infra/libs/cros/lab_inventory/utils"
)

// InventoryServerImpl implements service interfaces.
type InventoryServerImpl struct {
}

var (
	getHwidDataFunc     = hwid.GetHwidData
	getDeviceConfigFunc = deviceconfig.GetCachedConfig
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

// AddCrosDevices adds new Chrome OS devices to the inventory.
func (is *InventoryServerImpl) AddCrosDevices(ctx context.Context, req *api.AddCrosDevicesRequest) (resp *api.AddCrosDevicesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err = req.Validate(); err != nil {
		return nil, err
	}
	addingResults, err := datastore.AddDevices(ctx, req.Devices)
	if err != nil {
		return nil, errors.Annotate(err, "internal error").Tag(grpcutil.InternalTag).Err()
	}
	passedDevices := getPassedResults(*addingResults)
	failedDevices := getFailedResults(*addingResults, true)
	resp = &api.AddCrosDevicesResponse{
		PassedDevices: passedDevices,
		FailedDevices: failedDevices,
	}
	if len(failedDevices) > 0 {
		err = errors.Reason("failed to add some (or all) devices").Tag(grpcutil.UnknownTag).Err()
	}
	return resp, err
}

func getDeviceConfigData(ctx context.Context, extendedData []*api.ExtendedDeviceData) ([]*api.ExtendedDeviceData, []*api.DeviceOpResult) {
	// Start to retrieve device config data.
	devCfgIds := make([]*device.ConfigId, len(extendedData))
	for i, d := range extendedData {
		devCfgIds[i] = d.LabConfig.DeviceConfigId
	}
	devCfgs, err := getDeviceConfigFunc(ctx, devCfgIds)
	newExtendedData := make([]*api.ExtendedDeviceData, 0, len(extendedData))
	failedDevices := make([]*api.DeviceOpResult, 0, len(extendedData))
	for i := range devCfgs {
		if err == nil || err.(errors.MultiError)[i] == nil {
			extendedData[i].DeviceConfig = devCfgs[i].(*device.Config)
			newExtendedData = append(newExtendedData, extendedData[i])
		} else {
			failedDevices = append(failedDevices, &api.DeviceOpResult{
				Id:       extendedData[i].LabConfig.GetId().GetValue(),
				Hostname: utils.GetHostname(extendedData[i].LabConfig),
				ErrorMsg: err.(errors.MultiError)[i].Error(),
			})
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
		hwidData, err := getHwidDataFunc(ctx, labData.GetManufacturingId().GetValue(), secret)
		if err != nil {
			failedDevices = append(failedDevices, &api.DeviceOpResult{
				Id:       string(r.Entity.ID),
				Hostname: r.Entity.Hostname,
				ErrorMsg: err.Error(),
			})
			continue
		}
		extendedData = append(extendedData, &api.ExtendedDeviceData{
			LabConfig: &labData,
			HwidData: &api.HwidData{
				Sku:     hwidData.Sku,
				Variant: hwidData.Variant,
			},
			// TODO (guocb)Add manufactoring data.
			// TODO (guocb) Get dut state data.
		})
	}
	// Get device config in a batch.
	extendedData, moreFailedDevices := getDeviceConfigData(ctx, extendedData)
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
	var result []datastore.DeviceOpResult
	r := ([]datastore.DeviceOpResult)(datastore.GetDevicesByIds(ctx, devIds))
	result = append(r, datastore.GetDevicesByHostnames(ctx, hostnames)...)

	extendedData, moreFailedDevices := getExtendedDeviceData(ctx, datastore.DeviceOpResults(result).Passed())
	failedDevices := getFailedResults(result, false)
	failedDevices = append(failedDevices, moreFailedDevices...)

	resp = &api.GetCrosDevicesResponse{
		Data:          extendedData,
		FailedDevices: failedDevices,
	}
	return resp, nil
}

// UpdateDutsStatus updates selected Duts' status labels related to testing.
func (is *InventoryServerImpl) UpdateDutsStatus(ctx context.Context, req *api.UpdateDutsStatusRequest) (resp *api.UpdateDutsStatusResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()

	if err = req.Validate(); err != nil {
		return nil, err
	}
	updatingResults, err := datastore.UpdateDutsStatus(changehistory.Use(ctx, req.Reason), req.States)
	if err != nil {
		return nil, err
	}

	updatedDevices := getPassedResults(updatingResults)
	failedDevices := getFailedResults(updatingResults, false)
	resp = &api.UpdateDutsStatusResponse{
		UpdatedDevices: updatedDevices,
		FailedDevices:  failedDevices,
	}
	if len(failedDevices) > 0 {
		err = errors.Reason("failed to update some (or all) device state").Tag(grpcutil.UnknownTag).Err()
	}
	return resp, err
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
	updatingResults, err := datastore.UpdateDeviceSetup(changehistory.Use(ctx, req.Reason), req.Devices)
	if err != nil {
		return nil, err
	}

	updatedDevices := getPassedResults(updatingResults)
	failedDevices := getFailedResults(updatingResults, false)
	resp = &api.UpdateCrosDevicesSetupResponse{
		UpdatedDevices: updatedDevices,
		FailedDevices:  failedDevices,
	}
	if len(failedDevices) > 0 {
		err = errors.Reason("failed to update some (or all) devices").Tag(grpcutil.UnknownTag).Err()
	}
	return resp, err
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
	failedDevices := getFailedResults(deletingResults, false)
	resp = &api.DeleteCrosDevicesResponse{
		RemovedDevices: removedDevices,
		FailedDevices:  failedDevices,
	}
	if len(failedDevices) > 0 {
		err = errors.Reason("failed to remove some (or all) devices").Tag(grpcutil.UnknownTag).Err()
	}
	return resp, err
}
