// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"net/http"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/grpcutil"
	"go.chromium.org/luci/server/auth"
	"golang.org/x/net/context"

	luciconfig "go.chromium.org/luci/config"
	"go.chromium.org/luci/config/impl/remote"
	"go.chromium.org/luci/config/server/cfgclient/textproto"
	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"

	proto "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/datastore"
)

const defaultCfgService = "luci-config.appspot.com"

var (
	parsePlatformsFunc = configuration.ParsePlatformsFromFile
)

func (cs *FleetServerImpl) newCfgInterface(ctx context.Context) luciconfig.Interface {
	if cs.cfgInterfaceFactory != nil {
		return cs.cfgInterfaceFactory(ctx)
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

// ImportChromePlatforms imports the Chrome Platform in batch.
func (cs *FleetServerImpl) ImportChromePlatforms(ctx context.Context, req *api.ImportChromePlatformsRequest) (response *api.ImportChromePlatformsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()

	var platforms []*proto.ChromePlatform
	oldP := &crimsonconfig.Platforms{}
	switch req.LocalFilepath {
	case "":
		logging.Debugf(ctx, "Importing chrome platforms from luci-config")
		cfgInterface := cs.newCfgInterface(ctx)
		fetchedConfigs, err := cfgInterface.GetConfig(ctx, luciconfig.ServiceSet("machine-db-dev"), "platforms.cfg", false)
		if err != nil {
			return nil, err
		}
		logging.Debugf(ctx, "fetched configs: %#v", fetchedConfigs)
		resolver := textproto.Message(oldP)
		resolver.Resolve(fetchedConfigs)
	default:
		logging.Debugf(ctx, "Importing chrome platforms from local config file")
		oldP, err = parsePlatformsFunc(req.LocalFilepath)
		if err != nil {
			return nil, err
		}
	}
	platforms = configuration.ToChromePlatforms(oldP)
	logging.Debugf(ctx, "%d platforms in total", len(platforms))
	res, err := configuration.InsertChromePlatforms(ctx, platforms)
	if err != nil {
		return nil, err
	}
	return &api.ImportChromePlatformsResponse{
		Passed: toChromePlatformResult(res.Passed()),
		Failed: toChromePlatformResult(res.Failed()),
	}, err
}

func toChromePlatformResult(res datastore.OpResults) []*api.ChromePlatformResult {
	cpRes := make([]*api.ChromePlatformResult, len(res))
	for i, r := range res {
		errMsg := ""
		if r.Err != nil {
			errMsg = r.Err.Error()
		}
		cpRes[i] = &api.ChromePlatformResult{
			Platform: r.Data.(*proto.ChromePlatform),
			ErrorMsg: errMsg,
		}
	}
	return cpRes
}
