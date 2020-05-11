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
)

// CreateMachineLSE creates machineLSE entry in database.
func (fs *FleetServerImpl) CreateMachineLSE(ctx context.Context, req *api.CreateMachineLSERequest) (rsp *proto.MachineLSE, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return nil, err
}

// UpdateMachineLSE updates the machineLSE information in database.
func (fs *FleetServerImpl) UpdateMachineLSE(ctx context.Context, req *api.UpdateMachineLSERequest) (rsp *proto.MachineLSE, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return nil, err
}

// GetMachineLSE gets the machineLSE information from database.
func (fs *FleetServerImpl) GetMachineLSE(ctx context.Context, req *api.GetMachineLSERequest) (rsp *proto.MachineLSE, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return nil, err
}

// ListMachineLSEs list the machineLSEs information from database.
func (fs *FleetServerImpl) ListMachineLSEs(ctx context.Context, req *api.ListMachineLSEsRequest) (rsp *api.ListMachineLSEsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return nil, err
}

// DeleteMachineLSE deletes the machineLSE from database.
func (fs *FleetServerImpl) DeleteMachineLSE(ctx context.Context, req *api.DeleteMachineLSERequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return nil, err
}
