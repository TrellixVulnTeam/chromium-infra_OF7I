// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"errors"
	"fmt"

	empty "github.com/golang/protobuf/ptypes/empty"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/grpcutil"
	"golang.org/x/net/context"
	status "google.golang.org/genproto/googleapis/rpc/status"

	proto "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"

	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
)

// CreateMachine creates machine entry in database.
func (fs *FleetServerImpl) CreateMachine(ctx context.Context, req *api.CreateMachineRequest) (rsp *proto.Machine, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Machine.Name = req.MachineId
	machine, err := registration.CreateMachine(ctx, req.Machine)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machine.Name = util.AddPrefix(machineCollection, machine.Name)
	return machine, err
}

// UpdateMachine updates the machine information in database.
func (fs *FleetServerImpl) UpdateMachine(ctx context.Context, req *api.UpdateMachineRequest) (rsp *proto.Machine, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Machine.Name = util.RemovePrefix(req.Machine.Name)
	machine, err := registration.UpdateMachine(ctx, req.Machine)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machine.Name = util.AddPrefix(machineCollection, machine.Name)
	return machine, err
}

// GetMachine gets the machine information from database.
func (fs *FleetServerImpl) GetMachine(ctx context.Context, req *api.GetMachineRequest) (rsp *proto.Machine, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	machine, err := registration.GetMachine(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machine.Name = util.AddPrefix(machineCollection, machine.Name)
	return machine, err
}

// ListMachines list the machines information from database.
func (fs *FleetServerImpl) ListMachines(ctx context.Context, req *api.ListMachinesRequest) (rsp *api.ListMachinesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := registration.ListMachines(ctx, pageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, machine := range result {
		machine.Name = util.AddPrefix(machineCollection, machine.Name)
	}
	return &api.ListMachinesResponse{
		Machines:      result,
		NextPageToken: nextPageToken,
	}, nil
}

// DeleteMachine deletes the machine from database.
func (fs *FleetServerImpl) DeleteMachine(ctx context.Context, req *api.DeleteMachineRequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	err = registration.DeleteMachine(ctx, name)
	return &empty.Empty{}, err
}

// ImportMachines imports the machines from parent sources.
func (fs *FleetServerImpl) ImportMachines(ctx context.Context, req *api.ImportMachinesRequest) (rsp *status.Status, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	source := req.GetMachineDbSource()
	if source == nil {
		return emptyMachineDBSourceStatus, errors.New(emptyMachineDBSource)
	}
	if source.GetHost() == "" {
		return invalidHostInMachineDBSourceStatus, errors.New(invalidHostInMachineDBSource)
	}
	mdbClient, err := fs.newMachineDBInterfaceFactory(ctx, source.GetHost())
	if err != nil {
		return machineDBConnectionFailureStatus, errors.New(machineDBConnectionFailure)
	}
	logging.Debugf(ctx, "Querying machine-db to get the list of machines")
	resp, err := mdbClient.ListMachines(ctx, &crimson.ListMachinesRequest{})
	if err != nil {
		return machineDBServiceFailureStatus("ListMachines"), fmt.Errorf(machineDBServiceFailure, "ListMachines")
	}
	logging.Debugf(ctx, "Processing %d machines", len(resp.Machines))
	return successStatus, err
}
