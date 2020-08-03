// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	empty "github.com/golang/protobuf/ptypes/empty"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/grpcutil"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
	"golang.org/x/net/context"
	status "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/controller"
	"infra/unifiedfleet/app/util"
)

func verifyLSEPrototype(ctx context.Context, lse *ufspb.MachineLSE) error {
	if lse.GetChromeBrowserMachineLse() != nil {
		if !util.IsInBrowserLab(lse.GetMachineLsePrototype()) {
			return grpcStatus.Errorf(codes.InvalidArgument, "Prototype %s doesn't belong to browser lab", lse.GetMachineLsePrototype())
		}
		resp, err := controller.GetMachineLSEPrototype(ctx, lse.GetMachineLsePrototype())
		if err != nil {
			return err
		}
		for _, v := range resp.GetVirtualRequirements() {
			if v.GetVirtualType() == ufspb.VirtualType_VIRTUAL_TYPE_VM {
				c := lse.GetChromeBrowserMachineLse().GetVmCapacity()
				if c < v.GetMin() || c > v.GetMax() {
					return grpcStatus.Errorf(codes.InvalidArgument, "Prototype %s is not matched to the vm capacity %d", lse.GetMachineLsePrototype(), c)
				}
			}
		}
	}
	return nil
}

// CreateMachineLSE creates machineLSE entry in database.
func (fs *FleetServerImpl) CreateMachineLSE(ctx context.Context, req *ufsAPI.CreateMachineLSERequest) (rsp *ufspb.MachineLSE, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := verifyLSEPrototype(ctx, req.GetMachineLSE()); err != nil {
		return nil, err
	}
	req.MachineLSE.Name = req.MachineLSEId
	machineLSE, err := controller.CreateMachineLSE(ctx, req.MachineLSE, req.Machines, req.GetNetworkOption())
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machineLSE.Name = util.AddPrefix(util.MachineLSECollection, machineLSE.Name)
	return machineLSE, err
}

// UpdateMachineLSE updates the machineLSE information in database.
func (fs *FleetServerImpl) UpdateMachineLSE(ctx context.Context, req *ufsAPI.UpdateMachineLSERequest) (rsp *ufspb.MachineLSE, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.MachineLSE.Name = util.RemovePrefix(req.MachineLSE.Name)
	machineLSE, err := controller.UpdateMachineLSE(ctx, req.MachineLSE, req.Machines, req.GetNetworkOptions(), req.GetStates())
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machineLSE.Name = util.AddPrefix(util.MachineLSECollection, machineLSE.Name)
	return machineLSE, err
}

// GetMachineLSE gets the machineLSE information from database.
func (fs *FleetServerImpl) GetMachineLSE(ctx context.Context, req *ufsAPI.GetMachineLSERequest) (rsp *ufspb.MachineLSE, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	machineLSE, err := controller.GetMachineLSE(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machineLSE.Name = util.AddPrefix(util.MachineLSECollection, machineLSE.Name)
	return machineLSE, err
}

// ListMachineLSEs list the machineLSEs information from database.
func (fs *FleetServerImpl) ListMachineLSEs(ctx context.Context, req *ufsAPI.ListMachineLSEsRequest) (rsp *ufsAPI.ListMachineLSEsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListMachineLSEs(ctx, pageSize, req.PageToken, req.Filter, req.KeysOnly)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, machineLSE := range result {
		machineLSE.Name = util.AddPrefix(util.MachineLSECollection, machineLSE.Name)
	}
	return &ufsAPI.ListMachineLSEsResponse{
		MachineLSEs:   result,
		NextPageToken: nextPageToken,
	}, nil
}

// DeleteMachineLSE deletes the machineLSE from database.
func (fs *FleetServerImpl) DeleteMachineLSE(ctx context.Context, req *ufsAPI.DeleteMachineLSERequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	err = controller.DeleteMachineLSE(ctx, name)
	return &empty.Empty{}, err
}

// CreateRackLSE creates rackLSE entry in database.
func (fs *FleetServerImpl) CreateRackLSE(ctx context.Context, req *ufsAPI.CreateRackLSERequest) (rsp *ufspb.RackLSE, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.RackLSE.Name = req.RackLSEId
	rackLSE, err := controller.CreateRackLSE(ctx, req.RackLSE)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rackLSE.Name = util.AddPrefix(util.RackLSECollection, rackLSE.Name)
	return rackLSE, err
}

// UpdateRackLSE updates the rackLSE information in database.
func (fs *FleetServerImpl) UpdateRackLSE(ctx context.Context, req *ufsAPI.UpdateRackLSERequest) (rsp *ufspb.RackLSE, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.RackLSE.Name = util.RemovePrefix(req.RackLSE.Name)
	rackLSE, err := controller.UpdateRackLSE(ctx, req.RackLSE)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rackLSE.Name = util.AddPrefix(util.RackLSECollection, rackLSE.Name)
	return rackLSE, err
}

// GetRackLSE gets the rackLSE information from database.
func (fs *FleetServerImpl) GetRackLSE(ctx context.Context, req *ufsAPI.GetRackLSERequest) (rsp *ufspb.RackLSE, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	rackLSE, err := controller.GetRackLSE(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rackLSE.Name = util.AddPrefix(util.RackLSECollection, rackLSE.Name)
	return rackLSE, err
}

// ListRackLSEs list the rackLSEs information from database.
func (fs *FleetServerImpl) ListRackLSEs(ctx context.Context, req *ufsAPI.ListRackLSEsRequest) (rsp *ufsAPI.ListRackLSEsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListRackLSEs(ctx, pageSize, req.PageToken, req.Filter, req.KeysOnly)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, rackLSE := range result {
		rackLSE.Name = util.AddPrefix(util.RackLSECollection, rackLSE.Name)
	}
	return &ufsAPI.ListRackLSEsResponse{
		RackLSEs:      result,
		NextPageToken: nextPageToken,
	}, nil
}

// DeleteRackLSE deletes the rackLSE from database.
func (fs *FleetServerImpl) DeleteRackLSE(ctx context.Context, req *ufsAPI.DeleteRackLSERequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	err = controller.DeleteRackLSE(ctx, name)
	return &empty.Empty{}, err
}

// ImportMachineLSEs imports browser machines' LSE & related infos (e.g. IP)
func (fs *FleetServerImpl) ImportMachineLSEs(ctx context.Context, req *ufsAPI.ImportMachineLSEsRequest) (response *status.Status, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	source := req.GetMachineDbSource()
	if err := ufsAPI.ValidateMachineDBSource(source); err != nil {
		return nil, err
	}
	mdbClient, err := fs.newMachineDBInterfaceFactory(ctx, source.GetHost())
	if err != nil {
		return nil, machineDBConnectionFailureStatus.Err()
	}
	logging.Debugf(ctx, "Querying machine-db to list the physical hosts")
	hosts, err := mdbClient.ListPhysicalHosts(ctx, &crimson.ListPhysicalHostsRequest{})
	if err != nil {
		return nil, machineDBServiceFailureStatus("ListPhysicalHosts").Err()
	}
	if err := ufsAPI.ValidateResourceKey(hosts.GetHosts(), "Name"); err != nil {
		return nil, errors.Annotate(err, "hosts has invalid chars").Err()
	}
	vms, err := mdbClient.ListVMs(ctx, &crimson.ListVMsRequest{})
	if err != nil {
		return nil, machineDBServiceFailureStatus("ListVMs").Err()
	}
	if err := ufsAPI.ValidateResourceKey(vms.GetVms(), "Name"); err != nil {
		return nil, errors.Annotate(err, "vms has invalid chars").Err()
	}
	logging.Debugf(ctx, "Querying machine-db to list the machines")
	machines, err := mdbClient.ListMachines(ctx, &crimson.ListMachinesRequest{})
	if err != nil {
		return nil, machineDBServiceFailureStatus("ListMachines").Err()
	}
	if err := ufsAPI.ValidateResourceKey(machines.GetMachines(), "Name"); err != nil {
		return nil, errors.Annotate(err, "machines has invalid chars").Err()
	}
	pageSize := fs.getImportPageSize()
	res, err := controller.ImportMachineLSEs(ctx, hosts.GetHosts(), vms.GetVms(), machines.GetMachines(), pageSize)
	s := processImportDatastoreRes(res, err)
	if s.Err() != nil {
		return s.Proto(), s.Err()
	}
	return successStatus.Proto(), nil
}

// ImportOSMachineLSEs imports chromeos devices machine lses
func (fs *FleetServerImpl) ImportOSMachineLSEs(ctx context.Context, req *ufsAPI.ImportOSMachineLSEsRequest) (response *status.Status, err error) {
	source := req.GetMachineDbSource()
	if err := ufsAPI.ValidateMachineDBSource(source); err != nil {
		return nil, err
	}
	client, err := fs.newCrosInventoryInterfaceFactory(ctx, source.GetHost())
	if err != nil {
		return nil, crosInventoryConnectionFailureStatus.Err()
	}
	resp, err := client.ListCrosDevicesLabConfig(ctx, &invV2Api.ListCrosDevicesLabConfigRequest{})
	if err != nil {
		return nil, crosInventoryServiceFailureStatus("ListCrosDevicesLabConfig").Err()
	}
	pageSize := fs.getImportPageSize()
	res, err := controller.ImportOSMachineLSEs(ctx, resp.GetLabConfigs(), pageSize)
	s := processImportDatastoreRes(res, err)
	if s.Err() != nil {
		return s.Proto(), s.Err()
	}
	return successStatus.Proto(), nil
}
