// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

// This is the entrypoint for the Karte service in production and dev.
// Control is transferred here, inside the Docker container, when the
// application starts.

import (
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/config/server/cfgmodule"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"

	"infra/cros/karte/internal/frontend"
)

// Transfer control to the LUCI server
func main() {
	// See https://bugs.chromium.org/p/chromium/issues/detail?id=1242998 for details.
	// TODO(gregorynisbet): Remove this once new behavior is default.
	datastore.EnableSafeGet()

	modules := []module.Module{
		gaeemulation.NewModuleFromFlags(),
		cfgmodule.NewModuleFromFlags(),
	}

	server.Main(nil, modules, func(srv *server.Server) error {
		logging.Infof(srv.Context, "Starting server.")
		logging.Infof(srv.Context, "Installing Services.")
		frontend.InstallServices(srv.PRPC)

		logging.Infof(srv.Context, "Initialization finished.")
		return nil
	})
}
