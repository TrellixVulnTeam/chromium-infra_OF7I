// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"

	empty "github.com/golang/protobuf/ptypes/empty"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/grpcutil"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
	status "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/controller"
	"infra/unifiedfleet/app/external"
	"infra/unifiedfleet/app/util"
)

func verifyLSEPrototype(ctx context.Context, lse *ufspb.MachineLSE) error {
	if lse.GetChromeBrowserMachineLse() != nil {
		if !util.IsInBrowserZone(lse.GetMachineLsePrototype()) {
			return grpcStatus.Errorf(codes.InvalidArgument, "Prototype %s doesn't belong to browser lab", lse.GetMachineLsePrototype())
		}
		resp, err := controller.GetMachineLSEPrototype(ctx, lse.GetMachineLsePrototype())
		if err != nil {
			return grpcStatus.Errorf(codes.InvalidArgument, "Prototype %s doesn't exist", lse.GetMachineLsePrototype())
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
	req.MachineLSE.Name = util.FormatDHCPHostname(req.MachineLSEId)
	req.MachineLSE.Hostname = util.FormatDHCPHostname(req.MachineLSE.Hostname)
	req.NetworkOption = updateNetworkOpt(req.MachineLSE.GetVlan(), req.MachineLSE.GetIp(), req.GetNetworkOption())

	machineLSE, err := controller.CreateMachineLSE(ctx, req.MachineLSE, req.GetNetworkOption())
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
	req.MachineLSE.Name = util.FormatDHCPHostname(util.RemovePrefix(req.MachineLSE.Name))
	req.MachineLSE.Hostname = util.FormatDHCPHostname(req.MachineLSE.Hostname)
	nwOpt := req.GetNetworkOptions()[req.MachineLSE.Name]
	nwOpt = updateNetworkOpt(req.MachineLSE.GetVlan(), req.MachineLSE.GetIp(), nwOpt)
	if nwOpt != nil {
		machinelse := req.MachineLSE
		var err error
		if req.UpdateMask != nil && len(req.UpdateMask.Paths) > 0 {
			machinelse, err = controller.UpdateMachineLSE(ctx, req.MachineLSE, req.UpdateMask)
			if err != nil {
				return nil, err
			}
		}

		// If network_option.delete is enabled, ignore network_option.vlan and return directly
		if nwOpt.GetDelete() {
			if err = controller.DeleteMachineLSEHost(ctx, req.MachineLSE.Name); err != nil {
				return nil, err
			}
			machinelse, err = controller.GetMachineLSE(ctx, req.MachineLSE.Name)
			if err != nil {
				return nil, err
			}
		} else if nwOpt.GetVlan() != "" || nwOpt.GetIp() != "" {
			machinelse, err = controller.UpdateMachineLSEHost(ctx, req.MachineLSE.Name, nwOpt)
			if err != nil {
				return nil, err
			}
		}

		// https://aip.dev/122 - as per AIP guideline
		machinelse.Name = util.AddPrefix(util.MachineLSECollection, machinelse.Name)
		return machinelse, nil
	}

	machinelse, err := controller.UpdateMachineLSE(ctx, req.MachineLSE, req.UpdateMask)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machinelse.Name = util.AddPrefix(util.MachineLSECollection, machinelse.Name)
	return machinelse, err
}

// GetMachineLSE gets the machineLSE information from database.
func (fs *FleetServerImpl) GetMachineLSE(ctx context.Context, req *ufsAPI.GetMachineLSERequest) (rsp *ufspb.MachineLSE, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.FormatDHCPHostname(util.RemovePrefix(req.Name))
	machineLSE, err := controller.GetMachineLSE(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machineLSE.Name = util.AddPrefix(util.MachineLSECollection, machineLSE.Name)
	return machineLSE, err
}

// BatchGetMachineLSEs gets a batch of machineLSE information from database.
func (fs *FleetServerImpl) BatchGetMachineLSEs(ctx context.Context, req *ufsAPI.BatchGetMachineLSEsRequest) (rsp *ufsAPI.BatchGetMachineLSEsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	lses, err := controller.BatchGetMachineLSEs(ctx, util.FormatDHCPHostnames(util.FormatInputNames(req.GetNames())))
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, lse := range lses {
		lse.Name = util.AddPrefix(util.MachineLSECollection, lse.Name)
	}
	return &ufsAPI.BatchGetMachineLSEsResponse{
		MachineLses: lses,
	}, nil
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
	result, nextPageToken, err := controller.ListMachineLSEs(ctx, pageSize, req.PageToken, req.Filter, req.KeysOnly, req.Full)
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
	name := util.FormatDHCPHostname(util.RemovePrefix(req.Name))
	err = controller.DeleteMachineLSE(ctx, name)
	return &empty.Empty{}, err
}

func updateNetworkOpt(userVlan, ip string, nwOpt *ufsAPI.NetworkOption) *ufsAPI.NetworkOption {
	if userVlan == "" && ip == "" {
		return nwOpt
	}
	if nwOpt == nil {
		return &ufsAPI.NetworkOption{
			Vlan: userVlan,
			Ip:   ip,
		}
	}
	nwOpt.Vlan = userVlan
	nwOpt.Ip = ip
	return nwOpt
}

// CreateVM creates a vm entry in database.
func (fs *FleetServerImpl) CreateVM(ctx context.Context, req *ufsAPI.CreateVMRequest) (rsp *ufspb.VM, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Vm.Name = util.FormatDHCPHostname(util.RemovePrefix(req.Vm.Name))
	req.Vm.Hostname = util.FormatDHCPHostname(util.RemovePrefix(req.Vm.Hostname))
	req.Vm.MachineLseId = util.FormatDHCPHostname(req.Vm.MachineLseId)
	req.NetworkOption = updateNetworkOpt(req.Vm.GetVlan(), req.Vm.GetIp(), req.GetNetworkOption())
	vm, err := controller.CreateVM(ctx, req.GetVm(), req.GetNetworkOption())
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	vm.Name = util.AddPrefix(util.VMCollection, vm.Name)
	return vm, err
}

// UpdateVM updates the vm information in database.
func (fs *FleetServerImpl) UpdateVM(ctx context.Context, req *ufsAPI.UpdateVMRequest) (rsp *ufspb.VM, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Vm.Name = util.FormatDHCPHostname(util.RemovePrefix(req.Vm.Name))
	req.Vm.Hostname = util.FormatDHCPHostname(util.RemovePrefix(req.Vm.Hostname))
	req.Vm.MachineLseId = util.FormatDHCPHostname(req.Vm.MachineLseId)
	req.NetworkOption = updateNetworkOpt(req.Vm.GetVlan(), req.Vm.GetIp(), req.GetNetworkOption())
	if req.GetNetworkOption() != nil {
		vm := req.Vm
		var err error
		if req.UpdateMask != nil && len(req.UpdateMask.Paths) > 0 {
			vm, err = controller.UpdateVM(ctx, req.Vm, req.UpdateMask)
			if err != nil {
				return nil, err
			}
		}

		// If network_option.delete is enabled, ignore network_option.vlan and return directly
		if req.GetNetworkOption().GetDelete() {
			if err = controller.DeleteVMHost(ctx, req.Vm.Name); err != nil {
				return nil, err
			}
			vm, err = controller.GetVM(ctx, req.Vm.Name)
			if err != nil {
				return nil, err
			}
		} else if req.GetNetworkOption().GetVlan() != "" || req.GetNetworkOption().GetIp() != "" {
			vm, err = controller.UpdateVMHost(ctx, req.Vm.Name, req.GetNetworkOption())
			if err != nil {
				return nil, err
			}
		}

		// https://aip.dev/122 - as per AIP guideline
		vm.Name = util.AddPrefix(util.VMCollection, vm.Name)
		return vm, nil
	}

	vm, err := controller.UpdateVM(ctx, req.Vm, req.UpdateMask)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	vm.Name = util.AddPrefix(util.VMCollection, vm.Name)
	return vm, err
}

// DeleteVM deletes a VM from database.
func (fs *FleetServerImpl) DeleteVM(ctx context.Context, req *ufsAPI.DeleteVMRequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.FormatDHCPHostname(util.RemovePrefix(req.Name))
	err = controller.DeleteVM(ctx, name)
	return &empty.Empty{}, err
}

// GetVM gets the VM information from database.
func (fs *FleetServerImpl) GetVM(ctx context.Context, req *ufsAPI.GetVMRequest) (rsp *ufspb.VM, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.FormatDHCPHostname(util.RemovePrefix(req.Name))
	vm, err := controller.GetVM(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	vm.Name = util.AddPrefix(util.VMCollection, vm.Name)
	return vm, err
}

// BatchGetVMs gets a batch of vms from database.
func (fs *FleetServerImpl) BatchGetVMs(ctx context.Context, req *ufsAPI.BatchGetVMsRequest) (rsp *ufsAPI.BatchGetVMsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	vms, err := controller.BatchGetVMs(ctx, util.FormatDHCPHostnames(util.FormatInputNames(req.GetNames())))
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, v := range vms {
		v.Name = util.AddPrefix(util.VMCollection, v.Name)
	}
	return &ufsAPI.BatchGetVMsResponse{
		Vms: vms,
	}, nil
}

// ListVMs list the vms information from database.
func (fs *FleetServerImpl) ListVMs(ctx context.Context, req *ufsAPI.ListVMsRequest) (rsp *ufsAPI.ListVMsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListVMs(ctx, pageSize, req.PageToken, req.Filter, req.KeysOnly)
	if err != nil {
		return nil, err
	}
	return &ufsAPI.ListVMsResponse{
		Vms:           result,
		NextPageToken: nextPageToken,
	}, nil
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
	es, err := external.GetServerInterface(ctx)
	if err != nil {
		return nil, err
	}
	mdbClient, err := es.NewMachineDBInterfaceFactory(ctx, source.GetHost())
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
	platforms, err := mdbClient.ListPlatforms(ctx, &crimson.ListPlatformsRequest{})
	if err != nil {
		return nil, machineDBServiceFailureStatus("ListPlatforms").Err()
	}
	pageSize := fs.getImportPageSize()
	res, err := controller.ImportMachineLSEs(ctx, hosts.GetHosts(), vms.GetVms(), machines.GetMachines(), platforms.GetPlatforms(), pageSize)
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
	es, err := external.GetServerInterface(ctx)
	if err != nil {
		return nil, err
	}
	client, err := es.NewCrosInventoryInterfaceFactory(ctx, source.GetHost())
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
