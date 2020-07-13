// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"net/http"

	empty "github.com/golang/protobuf/ptypes/empty"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/grpcutil"
	"go.chromium.org/luci/server/auth"
	"golang.org/x/net/context"

	status "google.golang.org/genproto/googleapis/rpc/status"

	luciproto "go.chromium.org/luci/common/proto"
	luciconfig "go.chromium.org/luci/config"
	"go.chromium.org/luci/config/impl/remote"
	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"

	proto "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/controller"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/util"
)

const defaultCfgService = "luci-config.appspot.com"

var (
	parsePlatformsFunc = configuration.ParsePlatformsFromFile
)

func (fs *FleetServerImpl) newCfgInterface(ctx context.Context) luciconfig.Interface {
	if fs.cfgInterfaceFactory != nil {
		return fs.cfgInterfaceFactory(ctx)
	}
	cfgService := config.Get(ctx).LuciConfigService
	if cfgService == "" {
		cfgService = defaultCfgService
	}
	return remote.New(cfgService, false, func(ctx context.Context) (*http.Client, error) {
		t, err := auth.GetRPCTransport(ctx, auth.AsSelf)
		if err != nil {
			return nil, err
		}
		return &http.Client{Transport: t}, nil
	})
}

// CreateChromePlatform creates chromeplatform entry in database.
func (fs *FleetServerImpl) CreateChromePlatform(ctx context.Context, req *api.CreateChromePlatformRequest) (rsp *proto.ChromePlatform, err error) {
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
func (fs *FleetServerImpl) UpdateChromePlatform(ctx context.Context, req *api.UpdateChromePlatformRequest) (rsp *proto.ChromePlatform, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.ChromePlatform.Name = util.RemovePrefix(req.ChromePlatform.Name)
	chromeplatform, err := controller.UpdateChromePlatform(ctx, req.ChromePlatform)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	chromeplatform.Name = util.AddPrefix(util.ChromePlatformCollection, chromeplatform.Name)
	return chromeplatform, err
}

// GetChromePlatform gets the chromeplatform information from database.
func (fs *FleetServerImpl) GetChromePlatform(ctx context.Context, req *api.GetChromePlatformRequest) (rsp *proto.ChromePlatform, err error) {
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

// ListChromePlatforms list the chromeplatforms information from database.
func (fs *FleetServerImpl) ListChromePlatforms(ctx context.Context, req *api.ListChromePlatformsRequest) (rsp *api.ListChromePlatformsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListChromePlatforms(ctx, pageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, chromePlatform := range result {
		chromePlatform.Name = util.AddPrefix(util.ChromePlatformCollection, chromePlatform.Name)
	}
	return &api.ListChromePlatformsResponse{
		ChromePlatforms: result,
		NextPageToken:   nextPageToken,
	}, nil
}

// DeleteChromePlatform deletes the chromeplatform from database.
func (fs *FleetServerImpl) DeleteChromePlatform(ctx context.Context, req *api.DeleteChromePlatformRequest) (rsp *empty.Empty, err error) {
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
func (fs *FleetServerImpl) ImportChromePlatforms(ctx context.Context, req *api.ImportChromePlatformsRequest) (response *status.Status, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()

	var platforms []*proto.ChromePlatform
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
		cfgInterface := fs.newCfgInterface(ctx)
		fetchedConfigs, err := cfgInterface.GetConfig(ctx, luciconfig.ServiceSet(configSource.ConfigServiceName), configSource.FileName, false)
		if err != nil {
			return nil, configServiceFailureStatus.Err()
		}
		logging.Debugf(ctx, "fetched configs: %#v", fetchedConfigs)
		if err := luciproto.UnmarshalTextML(fetchedConfigs.Content, oldP); err != nil {
			return nil, invalidConfigFileContentStatus.Err()
		}
	}
	platforms = util.ToChromePlatforms(oldP)

	logging.Debugf(ctx, "Importing %d platforms", len(platforms))
	if err := api.ValidateResourceKey(platforms, "Name"); err != nil {
		return nil, err
	}
	res, err := controller.ImportChromePlatforms(ctx, platforms)
	s := processImportDatastoreRes(res, err)
	return s.Proto(), s.Err()
}

// CreateMachineLSEPrototype creates machinelseprototype entry in database.
func (fs *FleetServerImpl) CreateMachineLSEPrototype(ctx context.Context, req *api.CreateMachineLSEPrototypeRequest) (rsp *proto.MachineLSEPrototype, err error) {
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
func (fs *FleetServerImpl) UpdateMachineLSEPrototype(ctx context.Context, req *api.UpdateMachineLSEPrototypeRequest) (rsp *proto.MachineLSEPrototype, err error) {
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
func (fs *FleetServerImpl) GetMachineLSEPrototype(ctx context.Context, req *api.GetMachineLSEPrototypeRequest) (rsp *proto.MachineLSEPrototype, err error) {
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

// ListMachineLSEPrototypes list the machinelseprototypes information from database.
func (fs *FleetServerImpl) ListMachineLSEPrototypes(ctx context.Context, req *api.ListMachineLSEPrototypesRequest) (rsp *api.ListMachineLSEPrototypesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListMachineLSEPrototypes(ctx, pageSize, req.PageToken, req.Filter)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, machineLSEPrototype := range result {
		machineLSEPrototype.Name = util.AddPrefix(util.MachineLSEPrototypeCollection, machineLSEPrototype.Name)
	}
	return &api.ListMachineLSEPrototypesResponse{
		MachineLSEPrototypes: result,
		NextPageToken:        nextPageToken,
	}, nil
}

// DeleteMachineLSEPrototype deletes the machinelseprototype from database.
func (fs *FleetServerImpl) DeleteMachineLSEPrototype(ctx context.Context, req *api.DeleteMachineLSEPrototypeRequest) (rsp *empty.Empty, err error) {
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
func (fs *FleetServerImpl) CreateRackLSEPrototype(ctx context.Context, req *api.CreateRackLSEPrototypeRequest) (rsp *proto.RackLSEPrototype, err error) {
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
func (fs *FleetServerImpl) UpdateRackLSEPrototype(ctx context.Context, req *api.UpdateRackLSEPrototypeRequest) (rsp *proto.RackLSEPrototype, err error) {
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
func (fs *FleetServerImpl) GetRackLSEPrototype(ctx context.Context, req *api.GetRackLSEPrototypeRequest) (rsp *proto.RackLSEPrototype, err error) {
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

// ListRackLSEPrototypes list the racklseprototypes information from database.
func (fs *FleetServerImpl) ListRackLSEPrototypes(ctx context.Context, req *api.ListRackLSEPrototypesRequest) (rsp *api.ListRackLSEPrototypesResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListRackLSEPrototypes(ctx, pageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, rackLSEPrototype := range result {
		rackLSEPrototype.Name = util.AddPrefix(util.RackLSEPrototypeCollection, rackLSEPrototype.Name)
	}
	return &api.ListRackLSEPrototypesResponse{
		RackLSEPrototypes: result,
		NextPageToken:     nextPageToken,
	}, nil
}

// DeleteRackLSEPrototype deletes the racklseprototype from database.
func (fs *FleetServerImpl) DeleteRackLSEPrototype(ctx context.Context, req *api.DeleteRackLSEPrototypeRequest) (rsp *empty.Empty, err error) {
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
func (fs *FleetServerImpl) CreateVlan(ctx context.Context, req *api.CreateVlanRequest) (rsp *proto.Vlan, err error) {
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
func (fs *FleetServerImpl) UpdateVlan(ctx context.Context, req *api.UpdateVlanRequest) (rsp *proto.Vlan, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Vlan.Name = util.RemovePrefix(req.Vlan.Name)
	vlan, err := controller.UpdateVlan(ctx, req.Vlan)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	vlan.Name = util.AddPrefix(util.VlanCollection, vlan.Name)
	return vlan, err
}

// GetVlan gets the vlan information from database.
func (fs *FleetServerImpl) GetVlan(ctx context.Context, req *api.GetVlanRequest) (rsp *proto.Vlan, err error) {
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

// ListVlans list the vlans information from database.
func (fs *FleetServerImpl) ListVlans(ctx context.Context, req *api.ListVlansRequest) (rsp *api.ListVlansResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pageSize := util.GetPageSize(req.PageSize)
	result, nextPageToken, err := controller.ListVlans(ctx, pageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, vlan := range result {
		vlan.Name = util.AddPrefix(util.VlanCollection, vlan.Name)
	}
	return &api.ListVlansResponse{
		Vlans:         result,
		NextPageToken: nextPageToken,
	}, nil
}

// DeleteVlan deletes the vlan from database.
func (fs *FleetServerImpl) DeleteVlan(ctx context.Context, req *api.DeleteVlanRequest) (rsp *empty.Empty, err error) {
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
func (fs *FleetServerImpl) ImportVlans(ctx context.Context, req *api.ImportVlansRequest) (response *status.Status, err error) {
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
	cfgInterface := fs.newCfgInterface(ctx)
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
func (fs *FleetServerImpl) ImportOSVlans(ctx context.Context, req *api.ImportOSVlansRequest) (response *status.Status, err error) {
	source := req.GetMachineDbSource()
	if err := api.ValidateMachineDBSource(source); err != nil {
		return nil, err
	}
	sheetClient, err := fs.newSheetInterface(ctx)
	if err != nil {
		return nil, sheetConnectionFailureStatus.Err()
	}
	res, err := controller.ImportOSVlans(ctx, sheetClient, fs.getImportPageSize())
	s := processImportDatastoreRes(res, err)
	if s.Err() != nil {
		return s.Proto(), s.Err()
	}
	return successStatus.Proto(), nil
}
