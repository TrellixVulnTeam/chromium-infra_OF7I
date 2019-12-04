// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"go.chromium.org/luci/grpc/grpcutil"
	"golang.org/x/net/context"

	api "infra/appengine/cros/lab_inventory/api/v1"
)

// InventoryServerImpl implements service interfaces.
type InventoryServerImpl struct {
}

// AddCrosDevices adds new Chrome OS devices to the inventory.
func (is *InventoryServerImpl) AddCrosDevices(ctx context.Context, req *api.AddCrosDevicesRequest) (resp *api.AddCrosDevicesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err = req.Validate(); err != nil {
		return nil, err
	}
	return &api.AddCrosDevicesResponse{}, nil
}

// GetCrosDevices retrieves requested Chrome OS devices from the inventory.
func (is *InventoryServerImpl) GetCrosDevices(ctx context.Context, req *api.GetCrosDevicesRequest) (resp *api.GetCrosDevicesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	return &api.GetCrosDevicesResponse{}, nil
}

// UpdateDutsStatus updates selected Duts' status labels related to testing.
func (is *InventoryServerImpl) UpdateDutsStatus(ctx context.Context, req *api.UpdateDutsStatusRequest) (resp *api.UpdateDutsStatusResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	return &api.UpdateDutsStatusResponse{}, nil
}

// UpdateCrosDevicesSetup updates the selected Chrome OS devices setup data in
// the inventory.
func (is *InventoryServerImpl) UpdateCrosDevicesSetup(ctx context.Context, req *api.UpdateCrosDevicesSetupRequest) (resp *api.UpdateCrosDevicesSetupResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	return &api.UpdateCrosDevicesSetupResponse{}, nil
}

// DeleteCrosDevices delete the selelcted DUTs from testing scheduler.
func (is *InventoryServerImpl) DeleteCrosDevices(ctx context.Context, req *api.DeleteCrosDevicesRequest) (resp *api.DeleteCrosDevicesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	return &api.DeleteCrosDevicesResponse{}, nil
}
