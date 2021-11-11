// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// package main implements the App Engine based HTTP server to handle request
// to GoFindit
package main

import (
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	"go.chromium.org/luci/server/router"
)

func init() {
	// TODO (crbug.com/1242998): Remove when this becomes the default (~Jan 2022).
	datastore.EnableSafeGet()
}

func main() {
	modules := []module.Module{
		gaeemulation.NewModuleFromFlags(),
	}

	server.Main(nil, modules, func(srv *server.Server) error {
		srv.Routes.GET("/", router.MiddlewareChain{}, func(c *router.Context) {
			c.Writer.Write([]byte("Placeholder for GoFindit UI"))
		})

		return nil
	})
}
