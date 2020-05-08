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
	return nil, err
}

// UpdateChromePlatform updates the chromeplatform information in database.
func (fs *FleetServerImpl) UpdateChromePlatform(ctx context.Context, req *api.UpdateChromePlatformRequest) (rsp *proto.ChromePlatform, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return nil, err
}

// GetChromePlatform gets the chromeplatform information from database.
func (fs *FleetServerImpl) GetChromePlatform(ctx context.Context, req *api.GetChromePlatformRequest) (rsp *proto.ChromePlatform, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return nil, err
}

// ListChromePlatforms list the chromeplatforms information from database.
func (fs *FleetServerImpl) ListChromePlatforms(ctx context.Context, req *api.ListChromePlatformsRequest) (rsp *api.ListChromePlatformsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return nil, err
}

// DeleteChromePlatform deletes the chromeplatform from database.
func (fs *FleetServerImpl) DeleteChromePlatform(ctx context.Context, req *api.DeleteChromePlatformRequest) (rsp *empty.Empty, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return nil, err
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
	platforms = configuration.ToChromePlatforms(oldP)

	logging.Debugf(ctx, "Importing %d platforms", len(platforms))
	res, err := configuration.InsertChromePlatforms(ctx, platforms)
	s := processImportDatastoreRes(res, err)
	return s.Proto(), s.Err()
}
