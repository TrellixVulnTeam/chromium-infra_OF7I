// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	empty "github.com/golang/protobuf/ptypes/empty"
	"go.chromium.org/luci/grpc/grpcutil"
	"golang.org/x/net/context"

	proto "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/util"
)

// CreateMachineLSE creates machineLSE entry in database.
func (fs *FleetServerImpl) CreateMachineLSE(ctx context.Context, req *api.CreateMachineLSERequest) (rsp *proto.MachineLSE, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.MachineLSE.Name = req.MachineLSEId
	machineLSE, err := inventory.CreateMachineLSE(ctx, req.MachineLSE)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machineLSE.Name = util.AddPrefix(machineLSECollection, machineLSE.Name)
	return machineLSE, err
}

// UpdateMachineLSE updates the machineLSE information in database.
func (fs *FleetServerImpl) UpdateMachineLSE(ctx context.Context, req *api.UpdateMachineLSERequest) (rsp *proto.MachineLSE, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.MachineLSE.Name = util.RemovePrefix(req.MachineLSE.Name)
	machineLSE, err := inventory.UpdateMachineLSE(ctx, req.MachineLSE)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machineLSE.Name = util.AddPrefix(machineLSECollection, machineLSE.Name)
	return machineLSE, err
}

// GetMachineLSE gets the machineLSE information from database.
func (fs *FleetServerImpl) GetMachineLSE(ctx context.Context, req *api.GetMachineLSERequest) (rsp *proto.MachineLSE, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	machineLSE, err := inventory.GetMachineLSE(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machineLSE.Name = util.AddPrefix(machineLSECollection, machineLSE.Name)
	return machineLSE, err
}

// ListMachineLSEs list the machineLSEs information from database.
func (fs *FleetServerImpl) ListMachineLSEs(ctx context.Context, req *api.ListMachineLSEsRequest) (rsp *api.ListMachineLSEsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := inventory.ListMachineLSEs(ctx, pageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, machineLSE := range result {
		machineLSE.Name = util.AddPrefix(machineLSECollection, machineLSE.Name)
	}
	return &api.ListMachineLSEsResponse{
		MachineLSEs:   result,
		NextPageToken: nextPageToken,
	}, nil
}

// DeleteMachineLSE deletes the machineLSE from database.
func (fs *FleetServerImpl) DeleteMachineLSE(ctx context.Context, req *api.DeleteMachineLSERequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	err = inventory.DeleteMachineLSE(ctx, name)
	return &empty.Empty{}, err
}

// CreateRackLSE creates rackLSE entry in database.
func (fs *FleetServerImpl) CreateRackLSE(ctx context.Context, req *api.CreateRackLSERequest) (rsp *proto.RackLSE, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.RackLSE.Name = req.RackLSEId
	rackLSE, err := inventory.CreateRackLSE(ctx, req.RackLSE)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rackLSE.Name = util.AddPrefix(rackLSECollection, rackLSE.Name)
	return rackLSE, err
}

// UpdateRackLSE updates the rackLSE information in database.
func (fs *FleetServerImpl) UpdateRackLSE(ctx context.Context, req *api.UpdateRackLSERequest) (rsp *proto.RackLSE, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.RackLSE.Name = util.RemovePrefix(req.RackLSE.Name)
	rackLSE, err := inventory.UpdateRackLSE(ctx, req.RackLSE)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rackLSE.Name = util.AddPrefix(rackLSECollection, rackLSE.Name)
	return rackLSE, err
}

// GetRackLSE gets the rackLSE information from database.
func (fs *FleetServerImpl) GetRackLSE(ctx context.Context, req *api.GetRackLSERequest) (rsp *proto.RackLSE, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return nil, err
}

// ListRackLSEs list the rackLSEs information from database.
func (fs *FleetServerImpl) ListRackLSEs(ctx context.Context, req *api.ListRackLSEsRequest) (rsp *api.ListRackLSEsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return nil, err
}

// DeleteRackLSE deletes the rackLSE from database.
func (fs *FleetServerImpl) DeleteRackLSE(ctx context.Context, req *api.DeleteRackLSERequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return nil, err
}
