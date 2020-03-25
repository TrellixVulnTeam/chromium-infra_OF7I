// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"go.chromium.org/luci/common/logging"
	"golang.org/x/net/context"

	api "infra/appengine/unified-fleet/api/v1"
)

// ConfigurationServerImpl implements the configuration server interfaces.
type ConfigurationServerImpl struct {
}

// ImportChromePlatforms imports the Chrome Platform in batch.
func (fs *ConfigurationServerImpl) ImportChromePlatforms(ctx context.Context, req *api.ImportChromePlatformsRequest) (response *api.ImportChromePlatformsResponse, err error) {
	logging.Debugf(ctx, "Importing chrome platforms")
	return &api.ImportChromePlatformsResponse{}, err
}
