// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"

	empty "github.com/golang/protobuf/ptypes/empty"
	"go.chromium.org/luci/grpc/grpcutil"

	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/controller"
	"infra/unifiedfleet/app/util"
)

// CreateCachingService creates CachingService entry in database.
func (fs *FleetServerImpl) CreateCachingService(ctx context.Context, req *ufsAPI.CreateCachingServiceRequest) (rsp *ufspb.CachingService, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.CachingService.Name = req.CachingServiceId
	cs, err := controller.CreateCachingService(ctx, req.CachingService)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline.
	cs.Name = util.AddPrefix(util.CachingServiceCollection, cs.Name)
	return cs, err
}

// UpdateCachingService updates the CachingService information in database.
func (fs *FleetServerImpl) UpdateCachingService(ctx context.Context, req *ufsAPI.UpdateCachingServiceRequest) (rsp *ufspb.CachingService, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.CachingService.Name = util.RemovePrefix(req.CachingService.Name)
	cs, err := controller.UpdateCachingService(ctx, req.CachingService, req.UpdateMask)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline.
	cs.Name = util.AddPrefix(util.CachingServiceCollection, cs.Name)
	return cs, err
}

// GetCachingService gets the CachingService information from database.
func (fs *FleetServerImpl) GetCachingService(ctx context.Context, req *ufsAPI.GetCachingServiceRequest) (rsp *ufspb.CachingService, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return nil, nil
}

// ListCachingServices list the CachingServices information from database.
func (fs *FleetServerImpl) ListCachingServices(ctx context.Context, req *ufsAPI.ListCachingServicesRequest) (rsp *ufsAPI.ListCachingServicesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return nil, nil
}

// DeleteCachingService deletes the CachingService from database.
func (fs *FleetServerImpl) DeleteCachingService(ctx context.Context, req *ufsAPI.DeleteCachingServiceRequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}
