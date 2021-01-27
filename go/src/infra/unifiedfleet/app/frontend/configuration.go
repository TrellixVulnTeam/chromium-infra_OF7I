// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"

	empty "github.com/golang/protobuf/ptypes/empty"
	"go.chromium.org/luci/common/logging"
	luciproto "go.chromium.org/luci/common/proto"
	luciconfig "go.chromium.org/luci/config"
	"go.chromium.org/luci/grpc/grpcutil"
	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
	status "google.golang.org/genproto/googleapis/rpc/status"

	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/controller"
	"infra/unifiedfleet/app/external"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/util"
)

var (
	parsePlatformsFunc = configuration.ParsePlatformsFromFile
)

// CreateChromePlatform creates chromeplatform entry in database.
func (fs *FleetServerImpl) CreateChromePlatform(ctx context.Context, req *ufsAPI.CreateChromePlatformRequest) (rsp *ufspb.ChromePlatform, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.ChromePlatform.Name = req.ChromePlatformId
	chromeplatform, err := controller.CreateChromePlatform(ctx, req.ChromePlatform)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	chromeplatform.Name = util.AddPrefix(util.ChromePlatformCollection, chromeplatform.Name)
	return chromeplatform, err
}

// UpdateChromePlatform updates the chromeplatform information in database.
func (fs *FleetServerImpl) UpdateChromePlatform(ctx context.Context, req *ufsAPI.UpdateChromePlatformRequest) (rsp *ufspb.ChromePlatform, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.ChromePlatform.Name = util.RemovePrefix(req.ChromePlatform.Name)
	chromeplatform, err := controller.UpdateChromePlatform(ctx, req.ChromePlatform, req.UpdateMask)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	chromeplatform.Name = util.AddPrefix(util.ChromePlatformCollection, chromeplatform.Name)
	return chromeplatform, err
}

// GetChromePlatform gets the chromeplatform information from database.
func (fs *FleetServerImpl) GetChromePlatform(ctx context.Context, req *ufsAPI.GetChromePlatformRequest) (rsp *ufspb.ChromePlatform, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	chromePlatform, err := controller.GetChromePlatform(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	chromePlatform.Name = util.AddPrefix(util.ChromePlatformCollection, chromePlatform.Name)
	return chromePlatform, err
}

// BatchGetChromePlatforms gets chrome platforms from database.
func (fs *FleetServerImpl) BatchGetChromePlatforms(ctx context.Context, req *ufsAPI.BatchGetChromePlatformsRequest) (rsp *ufsAPI.BatchGetChromePlatformsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	platforms, err := controller.BatchGetChromePlatforms(ctx, util.FormatInputNames(req.GetNames()))
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, v := range platforms {
		v.Name = util.AddPrefix(util.ChromePlatformCollection, v.Name)
	}
	return &ufsAPI.BatchGetChromePlatformsResponse{
		ChromePlatforms: platforms,
	}, nil
}

// GetDHCPConfig gets a dhcp record from database.
func (fs *FleetServerImpl) GetDHCPConfig(ctx context.Context, req *ufsAPI.GetDHCPConfigRequest) (rsp *ufspb.DHCPConfig, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Hostname = util.FormatDHCPHostname(req.Hostname)
	dhcp, err := controller.GetDHCPConfig(ctx, req.GetHostname())
	if err != nil {
		return nil, err
	}
	return dhcp, err
}

// BatchGetDHCPConfigs gets a batch of dhcp records from database.
func (fs *FleetServerImpl) BatchGetDHCPConfigs(ctx context.Context, req *ufsAPI.BatchGetDHCPConfigsRequest) (rsp *ufsAPI.BatchGetDHCPConfigsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	dhcps, err := controller.BatchGetDhcpConfigs(ctx, util.FormatDHCPHostnames(util.FormatInputNames(req.GetNames())))
	if err != nil {
		return nil, err
	}
	return &ufsAPI.BatchGetDHCPConfigsResponse{
		DhcpConfigs: dhcps,
	}, nil
}

// ListChromePlatforms list the chromeplatforms information from database.
func (fs *FleetServerImpl) ListChromePlatforms(ctx context.Context, req *ufsAPI.ListChromePlatformsRequest) (rsp *ufsAPI.ListChromePlatformsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListChromePlatforms(ctx, pageSize, req.PageToken, req.Filter, req.KeysOnly)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, chromePlatform := range result {
		chromePlatform.Name = util.AddPrefix(util.ChromePlatformCollection, chromePlatform.Name)
	}
	return &ufsAPI.ListChromePlatformsResponse{
		ChromePlatforms: result,
		NextPageToken:   nextPageToken,
	}, nil
}

// DeleteChromePlatform deletes the chromeplatform from database.
func (fs *FleetServerImpl) DeleteChromePlatform(ctx context.Context, req *ufsAPI.DeleteChromePlatformRequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	err = controller.DeleteChromePlatform(ctx, name)
	return &empty.Empty{}, err
}

// ImportChromePlatforms imports the Chrome Platform in batch.
func (fs *FleetServerImpl) ImportChromePlatforms(ctx context.Context, req *ufsAPI.ImportChromePlatformsRequest) (response *status.Status, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()

	var platforms []*ufspb.ChromePlatform
	oldP := &crimsonconfig.Platforms{}
	configSource := req.GetConfigSource()
	if configSource == nil {
		return nil, emptyConfigSourceStatus.Err()
	}

	switch configSource.ConfigServiceName {
	case "":
		logging.Debugf(ctx, "Importing chrome platforms from local config file")
		oldP, err = parsePlatformsFunc(configSource.FileName)
		if err != nil {
			return nil, invalidConfigFileContentStatus.Err()
		}
	default:
		logging.Debugf(ctx, "Importing chrome platforms from luci-config")
		es, err := external.GetServerInterface(ctx)
		if err != nil {
			return nil, err
		}
		cfgInterface := es.NewCfgInterface(ctx)
		fetchedConfigs, err := cfgInterface.GetConfig(ctx, luciconfig.ServiceSet(configSource.ConfigServiceName), configSource.FileName, false)
		if err != nil {
			logging.Debugf(ctx, "Fail to fetch configs: %s", err.Error())
			return nil, configServiceFailureStatus.Err()
		}
		if err := luciproto.UnmarshalTextML(fetchedConfigs.Content, oldP); err != nil {
			logging.Debugf(ctx, "Fail to unmarshal configs: %s", err.Error())
			return nil, invalidConfigFileContentStatus.Err()
		}
	}
	platforms = util.ToChromePlatforms(oldP)

	logging.Debugf(ctx, "Importing %d platforms", len(platforms))
	if err := ufsAPI.ValidateResourceKey(platforms, "Name"); err != nil {
		return nil, err
	}
	res, err := controller.ImportChromePlatforms(ctx, platforms, fs.getImportPageSize())
	s := processImportDatastoreRes(res, err)
	return s.Proto(), s.Err()
}

// ListOSVersions lists the chrome os versions in batch.
func (fs *FleetServerImpl) ListOSVersions(ctx context.Context, req *ufsAPI.ListOSVersionsRequest) (response *ufsAPI.ListOSVersionsResponse, err error) {
	return nil, nil
}

// ImportOSVersions imports the Chrome OSVersion in batch.
func (fs *FleetServerImpl) ImportOSVersions(ctx context.Context, req *ufsAPI.ImportOSVersionsRequest) (response *status.Status, err error) {
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
	logging.Debugf(ctx, "Querying machine-db to get the list of oses")
	resp, err := mdbClient.ListOSes(ctx, &crimson.ListOSesRequest{})
	if err != nil {
		return nil, machineDBServiceFailureStatus("ListOSes").Err()
	}
	oses := util.ToOses(resp.GetOses())
	res, err := controller.ImportOSes(ctx, oses, fs.getImportPageSize())
	s := processImportDatastoreRes(res, err)
	if s.Err() != nil {
		return s.Proto(), s.Err()
	}
	return successStatus.Proto(), nil
}

// CreateMachineLSEPrototype creates machinelseprototype entry in database.
func (fs *FleetServerImpl) CreateMachineLSEPrototype(ctx context.Context, req *ufsAPI.CreateMachineLSEPrototypeRequest) (rsp *ufspb.MachineLSEPrototype, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.MachineLSEPrototype.Name = req.MachineLSEPrototypeId
	machineLSEPrototype, err := controller.CreateMachineLSEPrototype(ctx, req.MachineLSEPrototype)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machineLSEPrototype.Name = util.AddPrefix(util.MachineLSEPrototypeCollection, machineLSEPrototype.Name)
	return machineLSEPrototype, err
}

// UpdateMachineLSEPrototype updates the machinelseprototype information in database.
func (fs *FleetServerImpl) UpdateMachineLSEPrototype(ctx context.Context, req *ufsAPI.UpdateMachineLSEPrototypeRequest) (rsp *ufspb.MachineLSEPrototype, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.MachineLSEPrototype.Name = util.RemovePrefix(req.MachineLSEPrototype.Name)
	machineLSEPrototype, err := controller.UpdateMachineLSEPrototype(ctx, req.MachineLSEPrototype)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machineLSEPrototype.Name = util.AddPrefix(util.MachineLSEPrototypeCollection, machineLSEPrototype.Name)
	return machineLSEPrototype, err
}

// GetMachineLSEPrototype gets the machinelseprototype information from database.
func (fs *FleetServerImpl) GetMachineLSEPrototype(ctx context.Context, req *ufsAPI.GetMachineLSEPrototypeRequest) (rsp *ufspb.MachineLSEPrototype, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	machineLSEPrototype, err := controller.GetMachineLSEPrototype(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machineLSEPrototype.Name = util.AddPrefix(util.MachineLSEPrototypeCollection, machineLSEPrototype.Name)
	return machineLSEPrototype, err
}

// BatchGetMachineLSEPrototypes gets machine lse prototypes from database.
func (fs *FleetServerImpl) BatchGetMachineLSEPrototypes(ctx context.Context, req *ufsAPI.BatchGetMachineLSEPrototypesRequest) (rsp *ufsAPI.BatchGetMachineLSEPrototypesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	prototypes, err := controller.BatchGetMachineLSEPrototypes(ctx, util.FormatInputNames(req.GetNames()))
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, v := range prototypes {
		v.Name = util.AddPrefix(util.MachineLSEPrototypeCollection, v.Name)
	}
	return &ufsAPI.BatchGetMachineLSEPrototypesResponse{
		MachineLsePrototypes: prototypes,
	}, nil
}

// ListMachineLSEPrototypes list the machinelseprototypes information from database.
func (fs *FleetServerImpl) ListMachineLSEPrototypes(ctx context.Context, req *ufsAPI.ListMachineLSEPrototypesRequest) (rsp *ufsAPI.ListMachineLSEPrototypesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListMachineLSEPrototypes(ctx, pageSize, req.PageToken, req.Filter, req.KeysOnly)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, machineLSEPrototype := range result {
		machineLSEPrototype.Name = util.AddPrefix(util.MachineLSEPrototypeCollection, machineLSEPrototype.Name)
	}
	return &ufsAPI.ListMachineLSEPrototypesResponse{
		MachineLSEPrototypes: result,
		NextPageToken:        nextPageToken,
	}, nil
}

// DeleteMachineLSEPrototype deletes the machinelseprototype from database.
func (fs *FleetServerImpl) DeleteMachineLSEPrototype(ctx context.Context, req *ufsAPI.DeleteMachineLSEPrototypeRequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	err = controller.DeleteMachineLSEPrototype(ctx, name)
	return &empty.Empty{}, err
}

// CreateRackLSEPrototype creates racklseprototype entry in database.
func (fs *FleetServerImpl) CreateRackLSEPrototype(ctx context.Context, req *ufsAPI.CreateRackLSEPrototypeRequest) (rsp *ufspb.RackLSEPrototype, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.RackLSEPrototype.Name = req.RackLSEPrototypeId
	rackLSEPrototype, err := controller.CreateRackLSEPrototype(ctx, req.RackLSEPrototype)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rackLSEPrototype.Name = util.AddPrefix(util.RackLSEPrototypeCollection, rackLSEPrototype.Name)
	return rackLSEPrototype, err
}

// UpdateRackLSEPrototype updates the racklseprototype information in database.
func (fs *FleetServerImpl) UpdateRackLSEPrototype(ctx context.Context, req *ufsAPI.UpdateRackLSEPrototypeRequest) (rsp *ufspb.RackLSEPrototype, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.RackLSEPrototype.Name = util.RemovePrefix(req.RackLSEPrototype.Name)
	rackLSEPrototype, err := controller.UpdateRackLSEPrototype(ctx, req.RackLSEPrototype)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rackLSEPrototype.Name = util.AddPrefix(util.RackLSEPrototypeCollection, rackLSEPrototype.Name)
	return rackLSEPrototype, err
}

// GetRackLSEPrototype gets the racklseprototype information from database.
func (fs *FleetServerImpl) GetRackLSEPrototype(ctx context.Context, req *ufsAPI.GetRackLSEPrototypeRequest) (rsp *ufspb.RackLSEPrototype, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	rackLSEPrototype, err := controller.GetRackLSEPrototype(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rackLSEPrototype.Name = util.AddPrefix(util.RackLSEPrototypeCollection, rackLSEPrototype.Name)
	return rackLSEPrototype, err
}

// BatchGetRackLSEPrototypes gets rack lse prototypes from database.
func (fs *FleetServerImpl) BatchGetRackLSEPrototypes(ctx context.Context, req *ufsAPI.BatchGetRackLSEPrototypesRequest) (rsp *ufsAPI.BatchGetRackLSEPrototypesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	prototypes, err := controller.BatchGetRackLSEPrototypes(ctx, util.FormatInputNames(req.GetNames()))
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, v := range prototypes {
		v.Name = util.AddPrefix(util.RackLSEPrototypeCollection, v.Name)
	}
	return &ufsAPI.BatchGetRackLSEPrototypesResponse{
		RackLsePrototypes: prototypes,
	}, nil
}

// ListRackLSEPrototypes list the racklseprototypes information from database.
func (fs *FleetServerImpl) ListRackLSEPrototypes(ctx context.Context, req *ufsAPI.ListRackLSEPrototypesRequest) (rsp *ufsAPI.ListRackLSEPrototypesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListRackLSEPrototypes(ctx, pageSize, req.PageToken, req.Filter, req.KeysOnly)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, rackLSEPrototype := range result {
		rackLSEPrototype.Name = util.AddPrefix(util.RackLSEPrototypeCollection, rackLSEPrototype.Name)
	}
	return &ufsAPI.ListRackLSEPrototypesResponse{
		RackLSEPrototypes: result,
		NextPageToken:     nextPageToken,
	}, nil
}

// DeleteRackLSEPrototype deletes the racklseprototype from database.
func (fs *FleetServerImpl) DeleteRackLSEPrototype(ctx context.Context, req *ufsAPI.DeleteRackLSEPrototypeRequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	err = controller.DeleteRackLSEPrototype(ctx, name)
	return &empty.Empty{}, err
}

// CreateVlan creates vlan entry in database.
func (fs *FleetServerImpl) CreateVlan(ctx context.Context, req *ufsAPI.CreateVlanRequest) (rsp *ufspb.Vlan, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Vlan.Name = req.VlanId
	vlan, err := controller.CreateVlan(ctx, req.Vlan)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	vlan.Name = util.AddPrefix(util.VlanCollection, vlan.Name)
	return vlan, err
}

// UpdateVlan updates the vlan information in database.
func (fs *FleetServerImpl) UpdateVlan(ctx context.Context, req *ufsAPI.UpdateVlanRequest) (rsp *ufspb.Vlan, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Vlan.Name = util.RemovePrefix(req.Vlan.Name)
	vlan, err := controller.UpdateVlan(ctx, req.Vlan, req.UpdateMask)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	vlan.Name = util.AddPrefix(util.VlanCollection, vlan.Name)
	return vlan, err
}

// GetVlan gets the vlan information from database.
func (fs *FleetServerImpl) GetVlan(ctx context.Context, req *ufsAPI.GetVlanRequest) (rsp *ufspb.Vlan, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	vlan, err := controller.GetVlan(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	vlan.Name = util.AddPrefix(util.VlanCollection, vlan.Name)
	return vlan, err
}

// BatchGetVlans gets a batch of vlans from database.
func (fs *FleetServerImpl) BatchGetVlans(ctx context.Context, req *ufsAPI.BatchGetVlansRequest) (rsp *ufsAPI.BatchGetVlansResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	vlans, err := controller.BatchGetVlans(ctx, util.FormatInputNames(req.GetNames()))
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, v := range vlans {
		v.Name = util.AddPrefix(util.VlanCollection, v.Name)
	}
	return &ufsAPI.BatchGetVlansResponse{
		Vlans: vlans,
	}, nil
}

// ListVlans list the vlans information from database.
func (fs *FleetServerImpl) ListVlans(ctx context.Context, req *ufsAPI.ListVlansRequest) (rsp *ufsAPI.ListVlansResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListVlans(ctx, pageSize, req.PageToken, req.Filter, req.KeysOnly)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, vlan := range result {
		vlan.Name = util.AddPrefix(util.VlanCollection, vlan.Name)
	}
	return &ufsAPI.ListVlansResponse{
		Vlans:         result,
		NextPageToken: nextPageToken,
	}, nil
}

// DeleteVlan deletes the vlan from database.
func (fs *FleetServerImpl) DeleteVlan(ctx context.Context, req *ufsAPI.DeleteVlanRequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	name := util.RemovePrefix(req.Name)
	err = controller.DeleteVlan(ctx, name)
	return &empty.Empty{}, err
}

// ImportVlans imports vlans & all IP-related infos.
func (fs *FleetServerImpl) ImportVlans(ctx context.Context, req *ufsAPI.ImportVlansRequest) (response *status.Status, err error) {
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

	logging.Debugf(ctx, "Importing vlans from luci-config: %s", configSource.FileName)
	es, err := external.GetServerInterface(ctx)
	if err != nil {
		return nil, err
	}
	cfgInterface := es.NewCfgInterface(ctx)
	fetchedConfigs, err := cfgInterface.GetConfig(ctx, luciconfig.ServiceSet(configSource.ConfigServiceName), configSource.FileName, false)
	if err != nil {
		return nil, configServiceFailureStatus.Err()
	}
	vlans := &crimsonconfig.VLANs{}
	if err := luciproto.UnmarshalTextML(fetchedConfigs.Content, vlans); err != nil {
		return nil, invalidConfigFileContentStatus.Err()
	}

	pageSize := fs.getImportPageSize()
	res, err := controller.ImportVlans(ctx, vlans.GetVlan(), pageSize)
	s := processImportDatastoreRes(res, err)
	if s.Err() != nil {
		return s.Proto(), s.Err()
	}
	return successStatus.Proto(), nil
}

// ImportOSVlans imports the ChromeOS vlans, ips and dhcp configs.
func (fs *FleetServerImpl) ImportOSVlans(ctx context.Context, req *ufsAPI.ImportOSVlansRequest) (response *status.Status, err error) {
	es, err := external.GetServerInterface(ctx)
	if err != nil {
		return nil, err
	}
	sheetClient, err := es.NewSheetInterface(ctx)
	if err != nil {
		return nil, sheetConnectionFailureStatus.Err()
	}
	networkCfg := config.Get(ctx).GetCrosNetworkConfig()
	gitClient, err := es.NewGitInterface(ctx, networkCfg.GetGitilesHost(), networkCfg.GetProject(), networkCfg.GetBranch())
	if err != nil {
		return nil, gitConnectionFailureStatus.Err()
	}
	res, err := controller.ImportOSVlans(ctx, sheetClient, gitClient, fs.getImportPageSize())
	s := processImportDatastoreRes(res, err)
	if s.Err() != nil {
		return s.Proto(), s.Err()
	}
	return successStatus.Proto(), nil
}
