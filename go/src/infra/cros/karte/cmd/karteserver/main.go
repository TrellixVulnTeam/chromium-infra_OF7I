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
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	"go.chromium.org/luci/server/router"
)

// Transfer control to the LUCI server
func main() {
	modules := []module.Module{
		gaeemulation.NewModuleFromFlags(),
		cfgmodule.NewModuleFromFlags(),
	}

	server.Main(nil, modules, func(srv *server.Server) error {
		srv.Routes.GET("/hello-world", router.MiddlewareChain{}, func(ctx *router.Context) {
			logging.Debugf(ctx.Context, "Hello world. a2c29304-30e1-41a2-b85e-e7f85eef4fd9.")
			ctx.Writer.Write([]byte("Hello world. 4a9cd07f-6dd9-4d00-9f99-4086b58045cb."))
		})

		return nil
	})
}
