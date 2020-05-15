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

	luciconfig "go.chromium.org/luci/config"
	"go.chromium.org/luci/config/impl/remote"
	"go.chromium.org/luci/config/server/cfgclient/textproto"
	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"

	proto "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/config"
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
	chromeplatform, err := configuration.CreateChromePlatform(ctx, req.ChromePlatform)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	chromeplatform.Name = util.AddPrefix(chromePlatformCollection, chromeplatform.Name)
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
	chromeplatform, err := configuration.UpdateChromePlatform(ctx, req.ChromePlatform)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	chromeplatform.Name = util.AddPrefix(chromePlatformCollection, chromeplatform.Name)
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
	chromePlatform, err := configuration.GetChromePlatform(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	chromePlatform.Name = util.AddPrefix(chromePlatformCollection, chromePlatform.Name)
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
	result, nextPageToken, err := configuration.ListChromePlatforms(ctx, pageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, chromePlatform := range result {
		chromePlatform.Name = util.AddPrefix(chromePlatformCollection, chromePlatform.Name)
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
	err = configuration.DeleteChromePlatform(ctx, name)
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
		resolver := textproto.Message(oldP)
		resolver.Resolve(fetchedConfigs)
	}
	platforms = util.ToChromePlatforms(oldP)

	logging.Debugf(ctx, "Importing %d platforms", len(platforms))
	res, err := configuration.ImportChromePlatforms(ctx, platforms)
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
	machineLSEPrototype, err := configuration.CreateMachineLSEPrototype(ctx, req.MachineLSEPrototype)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machineLSEPrototype.Name = util.AddPrefix(machineLSEPrototypeCollection, machineLSEPrototype.Name)
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
	machineLSEPrototype, err := configuration.UpdateMachineLSEPrototype(ctx, req.MachineLSEPrototype)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machineLSEPrototype.Name = util.AddPrefix(machineLSEPrototypeCollection, machineLSEPrototype.Name)
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
	machineLSEPrototype, err := configuration.GetMachineLSEPrototype(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	machineLSEPrototype.Name = util.AddPrefix(machineLSEPrototypeCollection, machineLSEPrototype.Name)
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
	result, nextPageToken, err := configuration.ListMachineLSEPrototypes(ctx, pageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, machineLSEPrototype := range result {
		machineLSEPrototype.Name = util.AddPrefix(machineLSEPrototypeCollection, machineLSEPrototype.Name)
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
	err = configuration.DeleteMachineLSEPrototype(ctx, name)
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
	rackLSEPrototype, err := configuration.CreateRackLSEPrototype(ctx, req.RackLSEPrototype)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rackLSEPrototype.Name = util.AddPrefix(rackLSEPrototypeCollection, rackLSEPrototype.Name)
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
	rackLSEPrototype, err := configuration.UpdateRackLSEPrototype(ctx, req.RackLSEPrototype)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rackLSEPrototype.Name = util.AddPrefix(rackLSEPrototypeCollection, rackLSEPrototype.Name)
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
	rackLSEPrototype, err := configuration.GetRackLSEPrototype(ctx, name)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	rackLSEPrototype.Name = util.AddPrefix(rackLSEPrototypeCollection, rackLSEPrototype.Name)
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
	result, nextPageToken, err := configuration.ListRackLSEPrototypes(ctx, pageSize, req.PageToken)
	if err != nil {
		return nil, err
	}
	// https://aip.dev/122 - as per AIP guideline
	for _, rackLSEPrototype := range result {
		rackLSEPrototype.Name = util.AddPrefix(rackLSEPrototypeCollection, rackLSEPrototype.Name)
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
	err = configuration.DeleteRackLSEPrototype(ctx, name)
	return &empty.Empty{}, err
}
