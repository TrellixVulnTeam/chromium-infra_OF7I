// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/grpcutil"
	"golang.org/x/net/context"

	api "infra/appengine/unified-fleet/api/v1"
	"infra/libs/fleet/configuration"
	"infra/libs/fleet/datastore"
	fleet "infra/libs/fleet/protos/go"
)

// ConfigurationServerImpl implements the configuration server interfaces.
type ConfigurationServerImpl struct {
}

var (
	parsePlatformsFunc = configuration.ParsePlatformsFromFile
)

// ImportChromePlatforms imports the Chrome Platform in batch.
func (fs *ConfigurationServerImpl) ImportChromePlatforms(ctx context.Context, req *api.ImportChromePlatformsRequest) (response *api.ImportChromePlatformsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	var platforms []*fleet.ChromePlatform
	if req.LocalFilepath != "" {
		logging.Debugf(ctx, "Importing chrome platforms from local config file")
		oldP, err := parsePlatformsFunc(req.LocalFilepath)
		if err != nil {
			return nil, err
		}
		platforms = configuration.ToChromePlatforms(oldP)
	} else {
	}
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
			Platform: r.Data.(*fleet.ChromePlatform),
			ErrorMsg: errMsg,
		}
	}
	return cpRes
}
