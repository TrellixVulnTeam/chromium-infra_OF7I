// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	empty "github.com/golang/protobuf/ptypes/empty"
	"go.chromium.org/luci/common/logging"
	luciconfig "go.chromium.org/luci/config"
	"go.chromium.org/luci/config/server/cfgclient/textproto"
	"go.chromium.org/luci/grpc/grpcutil"
	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
	"golang.org/x/net/context"
	status "google.golang.org/genproto/googleapis/rpc/status"
	proto "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/controller"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/util"
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
	machine, err := controller.CreateMachine(ctx, req.Machine)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machine.Name = util.AddPrefix(util.MachineCollection, machine.Name)
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
	machine, err := controller.UpdateMachine(ctx, req.Machine)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machine.Name = util.AddPrefix(util.MachineCollection, machine.Name)
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
	machine, err := controller.GetMachine(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machine.Name = util.AddPrefix(util.MachineCollection, machine.Name)
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
	result, nextPageToken, err := controller.ListMachines(ctx, pageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, machine := range result {
		machine.Name = util.AddPrefix(util.MachineCollection, machine.Name)
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
	err = controller.DeleteMachine(ctx, name)
	return &empty.Empty{}, err
}

// ImportMachines imports the machines from parent sources.
func (fs *FleetServerImpl) ImportMachines(ctx context.Context, req *api.ImportMachinesRequest) (rsp *status.Status, err error) {
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
	logging.Debugf(ctx, "Querying machine-db to get the list of machines")
	resp, err := mdbClient.ListMachines(ctx, &crimson.ListMachinesRequest{})
	if err != nil {
		return nil, machineDBServiceFailureStatus("ListMachines").Err()
	}
	logging.Debugf(ctx, "Querying machine-db to get the list of nics")
	nics, err := mdbClient.ListNICs(ctx, &crimson.ListNICsRequest{})
	if err != nil {
		return nil, machineDBServiceFailureStatus("ListNICs").Err()
	}
	_, _, _, machineToNics, machineToDracs, machineToSwitch := util.ProcessNics(nics.Nics)
	logging.Debugf(ctx, "Importing %d machines", len(resp.Machines))
	pageSize := fs.getImportPageSize()
	machines := util.ToChromeMachines(resp.GetMachines(), machineToNics, machineToDracs, machineToSwitch)
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(machines))
		logging.Debugf(ctx, "importing %dth - %dth", i, end-1)
		res, err := controller.ImportMachines(ctx, machines[i:end])
		s := processImportDatastoreRes(res, err)
		if s.Err() != nil {
			return s.Proto(), s.Err()
		}
		if i+pageSize >= len(machines) {
			break
		}
	}

	return successStatus.Proto(), nil
}

// CreateRack creates rack entry in database.
func (fs *FleetServerImpl) CreateRack(ctx context.Context, req *api.CreateRackRequest) (rsp *proto.Rack, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Rack.Name = req.RackId
	rack, err := controller.CreateRack(ctx, req.Rack)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rack.Name = util.AddPrefix(util.RackCollection, rack.Name)
	return rack, err
}

// UpdateRack updates the rack information in database.
func (fs *FleetServerImpl) UpdateRack(ctx context.Context, req *api.UpdateRackRequest) (rsp *proto.Rack, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Rack.Name = util.RemovePrefix(req.Rack.Name)
	rack, err := controller.UpdateRack(ctx, req.Rack)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rack.Name = util.AddPrefix(util.RackCollection, rack.Name)
	return rack, err
}

// GetRack gets the rack information from database.
func (fs *FleetServerImpl) GetRack(ctx context.Context, req *api.GetRackRequest) (rsp *proto.Rack, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	rack, err := controller.GetRack(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rack.Name = util.AddPrefix(util.RackCollection, rack.Name)
	return rack, err
}

// ListRacks list the racks information from database.
func (fs *FleetServerImpl) ListRacks(ctx context.Context, req *api.ListRacksRequest) (rsp *api.ListRacksResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListRacks(ctx, pageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, rack := range result {
		rack.Name = util.AddPrefix(util.RackCollection, rack.Name)
	}
	return &api.ListRacksResponse{
		Racks:         result,
		NextPageToken: nextPageToken,
	}, nil
}

// DeleteRack deletes the rack from database.
func (fs *FleetServerImpl) DeleteRack(ctx context.Context, req *api.DeleteRackRequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	err = controller.DeleteRack(ctx, name)
	return &empty.Empty{}, err
}

// CreateNic creates nic entry in database.
func (fs *FleetServerImpl) CreateNic(ctx context.Context, req *api.CreateNicRequest) (rsp *proto.Nic, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Nic.Name = req.NicId
	nic, err := controller.CreateNic(ctx, req.Nic)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	nic.Name = util.AddPrefix(util.NicCollection, nic.Name)
	return nic, err
}

// UpdateNic updates the nic information in database.
func (fs *FleetServerImpl) UpdateNic(ctx context.Context, req *api.UpdateNicRequest) (rsp *proto.Nic, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Nic.Name = util.RemovePrefix(req.Nic.Name)
	nic, err := controller.UpdateNic(ctx, req.Nic)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	nic.Name = util.AddPrefix(util.NicCollection, nic.Name)
	return nic, err
}

// GetNic gets the nic information from database.
func (fs *FleetServerImpl) GetNic(ctx context.Context, req *api.GetNicRequest) (rsp *proto.Nic, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	nic, err := controller.GetNic(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	nic.Name = util.AddPrefix(util.NicCollection, nic.Name)
	return nic, err
}

// ListNics list the nics information from database.
func (fs *FleetServerImpl) ListNics(ctx context.Context, req *api.ListNicsRequest) (rsp *api.ListNicsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListNics(ctx, pageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, nic := range result {
		nic.Name = util.AddPrefix(util.NicCollection, nic.Name)
	}
	return &api.ListNicsResponse{
		Nics:          result,
		NextPageToken: nextPageToken,
	}, nil
}

// DeleteNic deletes the nic from database.
func (fs *FleetServerImpl) DeleteNic(ctx context.Context, req *api.DeleteNicRequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	err = controller.DeleteNic(ctx, name)
	return &empty.Empty{}, err
}

// ImportNics imports the nics info in batch.
func (fs *FleetServerImpl) ImportNics(ctx context.Context, req *api.ImportNicsRequest) (response *status.Status, err error) {
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
	logging.Debugf(ctx, "Querying machine-db to get the list of nics")
	resp, err := mdbClient.ListNICs(ctx, &crimson.ListNICsRequest{})
	if err != nil {
		return nil, machineDBServiceFailureStatus("ListMachines").Err()
	}
	pageSize := fs.getImportPageSize()
	newNics, newDracs, dhcps, _, _, _ := util.ProcessNics(resp.Nics)
	// Please note that the importing here is not in one transaction, which
	// actually may cause data incompleteness. But as the importing job
	// will be triggered periodically, such incompleteness that's caused by
	// potential failure will be ignored.
	logging.Debugf(ctx, "Importing %d nics", len(newNics))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(newNics))
		logging.Debugf(ctx, "importing nics %dth - %dth", i, end-1)
		res, err := controller.ImportNics(ctx, newNics[i:end])
		s := processImportDatastoreRes(res, err)
		if s.Err() != nil {
			return s.Proto(), s.Err()
		}
		if i+pageSize >= len(newNics) {
			break
		}
	}
	logging.Debugf(ctx, "Importing %d dracs", len(newDracs))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(newDracs))
		logging.Debugf(ctx, "importing dracs %dth - %dth", i, end-1)
		res, err := controller.ImportDracs(ctx, newDracs[i:end])
		s := processImportDatastoreRes(res, err)
		if s.Err() != nil {
			return s.Proto(), s.Err()
		}
		if i+pageSize >= len(newDracs) {
			break
		}
	}
	logging.Debugf(ctx, "Importing %d dhcps", len(dhcps))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(dhcps))
		logging.Debugf(ctx, "importing dhcps %dth - %dth", i, end-1)
		res, err := configuration.ImportDHCPConfigs(ctx, dhcps[i:end])
		s := processImportDatastoreRes(res, err)
		if s.Err() != nil {
			return s.Proto(), s.Err()
		}
		if i+pageSize >= len(dhcps) {
			break
		}
	}
	return successStatus.Proto(), nil
}

// ImportDatacenters imports the datacenter and its related info in batch.
func (fs *FleetServerImpl) ImportDatacenters(ctx context.Context, req *api.ImportDatacentersRequest) (response *status.Status, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	configSource := req.GetConfigSource()
	if configSource == nil {
		return nil, emptyConfigSourceStatus.Err()
	}
	if configSource.ConfigServiceName == "" {
		return nil, invalidConfigServiceName.Err()
	}

	logging.Debugf(ctx, "Importing datacenters from luci-config: %s", configSource.FileName)
	cfgInterface := fs.newCfgInterface(ctx)
	fetchedConfigs, err := cfgInterface.GetConfig(ctx, luciconfig.ServiceSet(configSource.ConfigServiceName), configSource.FileName, false)
	if err != nil {
		return nil, configServiceFailureStatus.Err()
	}
	dc := &crimsonconfig.Datacenter{}
	resolver := textproto.Message(dc)
	resolver.Resolve(fetchedConfigs)
	logging.Debugf(ctx, "processing datacenter: %s", dc.GetName())

	pageSize := fs.getImportPageSize()
	res, err := controller.ImportDatacenter(ctx, dc, pageSize)
	s := processImportDatastoreRes(res, err)
	if s.Err() != nil {
		return s.Proto(), s.Err()
	}
	return successStatus.Proto(), nil
}

// ImportDatacenterConfigs imports the datacenter configs
//
// It's not an exposed RPC.
func (fs *FleetServerImpl) ImportDatacenterConfigs(ctx context.Context) ([]string, error) {
	logging.Debugf(ctx, "Importing the default datacenter config file from luci-config: %s", datacenterConfigFile)
	cfgInterface := fs.newCfgInterface(ctx)
	c, err := cfgInterface.GetConfig(ctx, luciconfig.ServiceSet(DefaultMachineDBService), datacenterConfigFile, false)
	if err != nil {
		return nil, err
	}
	dcs := &crimsonconfig.Datacenters{}
	logging.Debugf(ctx, "%#v", c)
	resolver := textproto.Message(dcs)
	resolver.Resolve(c)
	return dcs.GetDatacenter(), nil
}

// CreateKVM creates kvm entry in database.
func (fs *FleetServerImpl) CreateKVM(ctx context.Context, req *api.CreateKVMRequest) (rsp *proto.KVM, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.KVM.Name = req.KVMId
	kvm, err := controller.CreateKVM(ctx, req.KVM)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	kvm.Name = util.AddPrefix(util.KVMCollection, kvm.Name)
	return kvm, err
}

// UpdateKVM updates the kvm information in database.
func (fs *FleetServerImpl) UpdateKVM(ctx context.Context, req *api.UpdateKVMRequest) (rsp *proto.KVM, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.KVM.Name = util.RemovePrefix(req.KVM.Name)
	kvm, err := controller.UpdateKVM(ctx, req.KVM)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	kvm.Name = util.AddPrefix(util.KVMCollection, kvm.Name)
	return kvm, err
}

// GetKVM gets the kvm information from database.
func (fs *FleetServerImpl) GetKVM(ctx context.Context, req *api.GetKVMRequest) (rsp *proto.KVM, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	kvm, err := controller.GetKVM(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	kvm.Name = util.AddPrefix(util.KVMCollection, kvm.Name)
	return kvm, err
}

// ListKVMs list the kvms information from database.
func (fs *FleetServerImpl) ListKVMs(ctx context.Context, req *api.ListKVMsRequest) (rsp *api.ListKVMsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListKVMs(ctx, pageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, kvm := range result {
		kvm.Name = util.AddPrefix(util.KVMCollection, kvm.Name)
	}
	return &api.ListKVMsResponse{
		KVMs:          result,
		NextPageToken: nextPageToken,
	}, nil
}

// DeleteKVM deletes the kvm from database.
func (fs *FleetServerImpl) DeleteKVM(ctx context.Context, req *api.DeleteKVMRequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	err = controller.DeleteKVM(ctx, name)
	return &empty.Empty{}, err
}

// CreateRPM creates rpm entry in database.
func (fs *FleetServerImpl) CreateRPM(ctx context.Context, req *api.CreateRPMRequest) (rsp *proto.RPM, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.RPM.Name = req.RPMId
	rpm, err := controller.CreateRPM(ctx, req.RPM)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rpm.Name = util.AddPrefix(util.RPMCollection, rpm.Name)
	return rpm, err
}

// UpdateRPM updates the rpm information in database.
func (fs *FleetServerImpl) UpdateRPM(ctx context.Context, req *api.UpdateRPMRequest) (rsp *proto.RPM, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.RPM.Name = util.RemovePrefix(req.RPM.Name)
	rpm, err := controller.UpdateRPM(ctx, req.RPM)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rpm.Name = util.AddPrefix(util.RPMCollection, rpm.Name)
	return rpm, err
}

// GetRPM gets the rpm information from database.
func (fs *FleetServerImpl) GetRPM(ctx context.Context, req *api.GetRPMRequest) (rsp *proto.RPM, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	rpm, err := controller.GetRPM(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rpm.Name = util.AddPrefix(util.RPMCollection, rpm.Name)
	return rpm, err
}

// ListRPMs list the rpms information from database.
func (fs *FleetServerImpl) ListRPMs(ctx context.Context, req *api.ListRPMsRequest) (rsp *api.ListRPMsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListRPMs(ctx, pageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, rpm := range result {
		rpm.Name = util.AddPrefix(util.RPMCollection, rpm.Name)
	}
	return &api.ListRPMsResponse{
		RPMs:          result,
		NextPageToken: nextPageToken,
	}, nil
}

// DeleteRPM deletes the rpm from database.
func (fs *FleetServerImpl) DeleteRPM(ctx context.Context, req *api.DeleteRPMRequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	err = controller.DeleteRPM(ctx, name)
	return &empty.Empty{}, err
}

// CreateDrac creates drac entry in database.
func (fs *FleetServerImpl) CreateDrac(ctx context.Context, req *api.CreateDracRequest) (rsp *proto.Drac, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Drac.Name = req.DracId
	drac, err := controller.CreateDrac(ctx, req.Drac)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	drac.Name = util.AddPrefix(util.DracCollection, drac.Name)
	return drac, err
}

// UpdateDrac updates the drac information in database.
func (fs *FleetServerImpl) UpdateDrac(ctx context.Context, req *api.UpdateDracRequest) (rsp *proto.Drac, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Drac.Name = util.RemovePrefix(req.Drac.Name)
	drac, err := controller.UpdateDrac(ctx, req.Drac)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	drac.Name = util.AddPrefix(util.DracCollection, drac.Name)
	return drac, err
}

// GetDrac gets the drac information from database.
func (fs *FleetServerImpl) GetDrac(ctx context.Context, req *api.GetDracRequest) (rsp *proto.Drac, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	drac, err := controller.GetDrac(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	drac.Name = util.AddPrefix(util.DracCollection, drac.Name)
	return drac, err
}

// ListDracs list the dracs information from database.
func (fs *FleetServerImpl) ListDracs(ctx context.Context, req *api.ListDracsRequest) (rsp *api.ListDracsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListDracs(ctx, pageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, drac := range result {
		drac.Name = util.AddPrefix(util.DracCollection, drac.Name)
	}
	return &api.ListDracsResponse{
		Dracs:         result,
		NextPageToken: nextPageToken,
	}, nil
}

// DeleteDrac deletes the drac from database.
func (fs *FleetServerImpl) DeleteDrac(ctx context.Context, req *api.DeleteDracRequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	err = controller.DeleteDrac(ctx, name)
	return &empty.Empty{}, err
}

// CreateSwitch creates switch entry in database.
func (fs *FleetServerImpl) CreateSwitch(ctx context.Context, req *api.CreateSwitchRequest) (rsp *proto.Switch, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Switch.Name = req.SwitchId
	s, err := controller.CreateSwitch(ctx, req.Switch)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	s.Name = util.AddPrefix(util.SwitchCollection, s.Name)
	return s, err
}

// UpdateSwitch updates the switch information in database.
func (fs *FleetServerImpl) UpdateSwitch(ctx context.Context, req *api.UpdateSwitchRequest) (rsp *proto.Switch, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Switch.Name = util.RemovePrefix(req.Switch.Name)
	s, err := controller.UpdateSwitch(ctx, req.Switch)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	s.Name = util.AddPrefix(util.SwitchCollection, s.Name)
	return s, err
}

// GetSwitch gets the switch information from database.
func (fs *FleetServerImpl) GetSwitch(ctx context.Context, req *api.GetSwitchRequest) (rsp *proto.Switch, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	s, err := controller.GetSwitch(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	s.Name = util.AddPrefix(util.SwitchCollection, s.Name)
	return s, err
}

// ListSwitches list the switches information from database.
func (fs *FleetServerImpl) ListSwitches(ctx context.Context, req *api.ListSwitchesRequest) (rsp *api.ListSwitchesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListSwitches(ctx, pageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, s := range result {
		s.Name = util.AddPrefix(util.SwitchCollection, s.Name)
	}
	return &api.ListSwitchesResponse{
		Switches:      result,
		NextPageToken: nextPageToken,
	}, nil
}

// DeleteSwitch deletes the switch from database.
func (fs *FleetServerImpl) DeleteSwitch(ctx context.Context, req *api.DeleteSwitchRequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	err = controller.DeleteSwitch(ctx, name)
	return &empty.Empty{}, err
}
