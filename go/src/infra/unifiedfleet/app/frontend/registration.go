// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"strings"

	empty "github.com/golang/protobuf/ptypes/empty"
	"go.chromium.org/luci/grpc/grpcutil"
	"golang.org/x/net/context"

	"infra/libs/fleet/registration"
	proto "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
)

// CreateMachine creates machine entry in database.
func (fs *FleetServerImpl) CreateMachine(ctx context.Context, req *api.CreateMachineRequest) (rsp *proto.Machine, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Machine.Name = strings.TrimSpace(req.Machine.GetName())
	if req.Machine.GetName() == "" {
		req.Machine.Name = strings.TrimSpace(req.MachineId)
	}
	return registration.CreateMachine(ctx, req.Machine)
}

// UpdateMachine updates the machine information in database.
func (fs *FleetServerImpl) UpdateMachine(ctx context.Context, req *api.UpdateMachineRequest) (rsp *proto.Machine, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Machine.Name = strings.TrimSpace(req.Machine.GetName())
	return registration.UpdateMachine(ctx, req.Machine)
}

// GetMachine gets the machine information from database.
func (fs *FleetServerImpl) GetMachine(ctx context.Context, req *api.GetMachineRequest) (rsp *proto.Machine, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return registration.GetMachine(ctx, strings.TrimSpace(req.Name))
}

// ListMachines list all the machines information from database.
func (fs *FleetServerImpl) ListMachines(ctx context.Context, req *api.ListMachinesRequest) (rsp *api.ListMachinesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	return nil, err
}

// DeleteMachine deletes the machine from database.
func (fs *FleetServerImpl) DeleteMachine(ctx context.Context, req *api.DeleteMachineRequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	err = registration.DeleteMachine(ctx, strings.TrimSpace(req.Name))
	return &empty.Empty{}, err
}
