// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Frontend service handles home page, API and redirects. Static files are
// served by GAE directly (configured in app.yaml).
package main

import (
	"net/http"

	"go.chromium.org/luci/appengine/gaemiddleware"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/config/server/cfgmodule"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	"go.chromium.org/luci/server/router"

	"infra/appengine/cr-rev/config"
)

func handleStartup(c *router.Context) {
	c.Writer.Write([]byte("OK"))
}

func main() {
	mw := router.MiddlewareChain{}
	cron := router.NewMiddlewareChain(gaemiddleware.RequireCron)
	modules := []module.Module{
		cfgmodule.NewModuleFromFlags(),
		gaeemulation.NewModuleFromFlags(),
	}

	server.Main(nil, modules, func(srv *server.Server) error {
		srv.Routes.GET("/_ah/start", mw, handleStartup)
		srv.Routes.GET("/internal/cron/import-config", cron, func(c *router.Context) {
			if err := config.Set(c.Context); err != nil {
				errors.Log(c.Context, err)
			}
			c.Writer.WriteHeader(http.StatusOK)
		})
		return nil
	})
}
