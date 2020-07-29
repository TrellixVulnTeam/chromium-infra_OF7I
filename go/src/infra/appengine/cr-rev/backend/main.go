// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Frontend service handles home page, API and redirects. Static files are
// served by GAE directly (configured in app.yaml).
package main

import (
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/module"
	"go.chromium.org/luci/server/router"
)

func handleStartup(c *router.Context) {
	c.Writer.Write([]byte("OK"))
}

func main() {
	mw := router.MiddlewareChain{}

	server.Main(nil, []module.Module{}, func(srv *server.Server) error {
		srv.Routes.GET("/_ah/start", mw, handleStartup)
		return nil
	})
}
