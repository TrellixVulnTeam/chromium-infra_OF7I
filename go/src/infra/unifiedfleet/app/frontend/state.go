// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/grpcutil"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
	status "google.golang.org/genproto/googleapis/rpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/controller"
)

// ImportStates imports states of crimson objects.
func (fs *FleetServerImpl) ImportStates(ctx context.Context, req *api.ImportStatesRequest) (response *status.Status, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	source := req.GetMachineDbSource()
	if err := api.ValidateMachineDBSource(source); err != nil {
		return nil, err
	}
	mdbClient, err := fs.newMachineDBInterfaceFactory(ctx, source.GetHost())
	if err != nil {
		return nil, machineDBConnectionFailureStatus.Err()
	}
	// Skip importing the states of racks, kvms, switches, and vlans, as their states should
	// be referenced by the states of their related racks & machines.
	logging.Debugf(ctx, "Querying machine-db to list the machines")
	machines, err := mdbClient.ListMachines(ctx, &crimson.ListMachinesRequest{})
	if err != nil {
		return nil, machineDBServiceFailureStatus("ListMachines").Err()
	}
	logging.Debugf(ctx, "Querying machine-db to list the vms")
	vms, err := mdbClient.ListVMs(ctx, &crimson.ListVMsRequest{})
	if err != nil {
		return nil, machineDBServiceFailureStatus("ListVMs").Err()
	}

	pageSize := fs.getImportPageSize()
	res, err := controller.ImportStates(ctx, machines.GetMachines(), vms.GetVms(), pageSize)
	s := processImportDatastoreRes(res, err)
	if s.Err() != nil {
		return s.Proto(), s.Err()
	}
	return successStatus.Proto(), nil
}

// UpdateState updates the state for a resource.
func (fs *FleetServerImpl) UpdateState(ctx context.Context, req *api.UpdateStateRequest) (response *ufspb.StateRecord, err error) {
	return nil, nil
}

// GetState returns the state for a resource.
func (fs *FleetServerImpl) GetState(ctx context.Context, req *api.GetStateRequest) (response *ufspb.StateRecord, err error) {
	return nil, nil
}
