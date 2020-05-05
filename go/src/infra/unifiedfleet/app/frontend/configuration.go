// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"net/http"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/grpcutil"
	"go.chromium.org/luci/server/auth"
	"golang.org/x/net/context"
	code "google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	status "google.golang.org/genproto/googleapis/rpc/status"

	"go.chromium.org/luci/common/errors"
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

// Error messages for configuration import
var (
	successStatus = &status.Status{
		Code: int32(code.Code_OK),
	}
	emptyConfigSource       = "Invalid argument - Config source is empty"
	emptyConfigSourceStatus = &status.Status{
		Code:    int32(code.Code_INVALID_ARGUMENT),
		Message: emptyConfigSource,
	}
	invalidConfigFileContent       = "The config file format is invalid"
	invalidConfigFileContentStatus = &status.Status{
		Code:    int32(code.Code_FAILED_PRECONDITION),
		Message: invalidConfigFileContent,
	}
	configServiceFailure       = "Fail to get configs from luci config service"
	configServiceFailureStatus = &status.Status{
		Code:    int32(code.Code_INTERNAL),
		Message: configServiceFailure,
	}
	insertDatastoreFailure       = "Fail to insert chrome platforms into datastore in importing"
	insertDatastoreFailureStatus = &status.Status{
		Code:    int32(code.Code_INTERNAL),
		Message: insertDatastoreFailure,
	}
)

// ImportChromePlatforms imports the Chrome Platform in batch.
func (cs *FleetServerImpl) ImportChromePlatforms(ctx context.Context, req *api.ImportChromePlatformsRequest) (response *status.Status, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()

	var platforms []*proto.ChromePlatform
	oldP := &crimsonconfig.Platforms{}
	configSource := req.GetConfigSource()
	if configSource == nil {
		return emptyConfigSourceStatus, errors.New(emptyConfigSource)
	}

	switch configSource.ConfigServiceName {
	case "":
		logging.Debugf(ctx, "Importing chrome platforms from local config file")
		oldP, err = parsePlatformsFunc(configSource.FileName)
		if err != nil {
			return invalidConfigFileContentStatus, errors.Annotate(err, invalidConfigFileContent).Err()
		}
	default:
		logging.Debugf(ctx, "Importing chrome platforms from luci-config")
		cfgInterface := cs.newCfgInterface(ctx)
		fetchedConfigs, err := cfgInterface.GetConfig(ctx, luciconfig.ServiceSet(configSource.ConfigServiceName), configSource.FileName, false)
		if err != nil {
			return configServiceFailureStatus, errors.Annotate(err, configServiceFailure).Err()
		}
		logging.Debugf(ctx, "fetched configs: %#v", fetchedConfigs)
		resolver := textproto.Message(oldP)
		resolver.Resolve(fetchedConfigs)
	}
	platforms = configuration.ToChromePlatforms(oldP)

	logging.Debugf(ctx, "Importing %d platforms", len(platforms))
	res, err := configuration.InsertChromePlatforms(ctx, platforms)
	if fails := res.Failed(); len(fails) > 0 {
		insertDatastoreFailureStatus.Details = errorToAny(fails)
		return insertDatastoreFailureStatus, err
	}
	return successStatus, nil
}

func errorToAny(res []*datastore.OpResult) []*any.Any {
	anys := make([]*any.Any, 0)
	for _, r := range res {
		e := &errdetails.ErrorInfo{
			Reason: r.Err.Error(),
		}
		any, err := ptypes.MarshalAny(e)
		if err != nil {
			continue
		}
		anys = append(anys, any)
	}
	return anys
}
