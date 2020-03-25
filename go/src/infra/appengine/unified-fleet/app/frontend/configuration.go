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
)

// ConfigurationServerImpl implements the configuration server interfaces.
type ConfigurationServerImpl struct {
}

// ImportChromePlatforms imports the Chrome Platform in batch.
func (fs *ConfigurationServerImpl) ImportChromePlatforms(ctx context.Context, req *api.ImportChromePlatformsRequest) (response *api.ImportChromePlatformsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	logging.Debugf(ctx, "Importing chrome platforms")
	if req.LocalFilepath != "" {
		platforms, err := configuration.ParsePlatformsFromFile(req.LocalFilepath)
		if err != nil {
			return nil, err
		}
		// TODO (xixuan): log first, save to datastore in next CL.
		logging.Debugf(ctx, "%s", platforms)
	}
	return &api.ImportChromePlatformsResponse{}, err
}
