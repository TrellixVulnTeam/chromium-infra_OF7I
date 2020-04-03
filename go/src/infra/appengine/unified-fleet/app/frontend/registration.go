// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"go.chromium.org/luci/grpc/grpcutil"
	"golang.org/x/net/context"

	api "infra/appengine/unified-fleet/api/v1"
)

// RegistrationServerImpl implements fleet interfaces.
type RegistrationServerImpl struct {
}

// CreateMachines creates machines in datastore
func (fs *RegistrationServerImpl) CreateMachines(ctx context.Context, req *api.MachineList) (response *api.MachineResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	return &api.MachineResponse{}, err
}

// GetMachines gets the machines information from datastore.
func (fs *RegistrationServerImpl) GetMachines(ctx context.Context, req *api.EntityIDList) (response *api.MachineResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	return &api.MachineResponse{}, err
}

// ListMachines gets all the machines information from datastore.
func (fs *RegistrationServerImpl) ListMachines(ctx context.Context, req *api.ListMachinesRequest) (response *api.MachineResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	return &api.MachineResponse{}, err
}

// UpdateMachines updates the machines information in datastore.
func (fs *RegistrationServerImpl) UpdateMachines(ctx context.Context, req *api.MachineList) (response *api.MachineResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	return &api.MachineResponse{}, err
}

// DeleteMachines deletes the machines from datastore.
func (fs *RegistrationServerImpl) DeleteMachines(ctx context.Context, req *api.EntityIDList) (response *api.EntityIDResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	return &api.EntityIDResponse{}, err
}
