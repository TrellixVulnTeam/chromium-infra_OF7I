// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"

	"infra/appengine/unified-fleet/app/config"
	"infra/appengine/unified-fleet/app/frontend"
)

func main() {
	modules := []module.Module{
		gaeemulation.NewModuleFromFlags(),
	}
	server.Main(nil, modules, func(srv *server.Server) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		srv.Context = config.Use(srv.Context, cfg)
		frontend.InstallServices(srv.PRPC)
		return nil
	})
}
