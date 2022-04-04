// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"go.chromium.org/luci/config/server/cfgmodule"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"

	"infra/appengine/poros/api/asset"
)

func init() {
	// TODO (crbug.com/1242998): Remove when this becomes the default (~Jan 2022).
	datastore.EnableSafeGet()
}

func main() {
	modules := []module.Module{
		cfgmodule.NewModuleFromFlags(),
		gaeemulation.NewModuleFromFlags(),
	}
	server.Main(nil, modules, func(srv *server.Server) error {
		assetHandler := &asset.AssetHandler{}
		asset.RegisterAssetServer(srv.PRPC, assetHandler)
		return nil
	})
}
