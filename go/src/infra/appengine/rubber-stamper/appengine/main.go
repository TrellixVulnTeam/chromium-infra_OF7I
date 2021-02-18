// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"go.chromium.org/luci/appengine/gaemiddleware"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/config/server/cfgmodule"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/tq"

	"infra/appengine/rubber-stamper/cron"
	"infra/appengine/rubber-stamper/internal/gerrit"
	"infra/appengine/rubber-stamper/internal/util"
)

func main() {
	modules := []module.Module{
		cfgmodule.NewModuleFromFlags(),
		gaeemulation.NewModuleFromFlags(),
		tq.NewModuleFromFlags(),
	}

	server.Main(nil, modules, func(srv *server.Server) error {
		var err error
		srv.Context = gerrit.Setup(srv.Context)
		srv.Context, err = util.SetupErrorReporting(srv.Context)
		if err != nil {
			logging.Errorf(srv.Context, "failed to set up ErrorReporting client")
		}

		basemw := router.NewMiddlewareChain()
		srv.Routes.GET("/hello-world", router.MiddlewareChain{}, func(c *router.Context) {
			logging.Debugf(c.Context, "Hello world")
			c.Writer.Write([]byte("Hello, world. This is Rubber-Stamper."))
		})

		srv.Routes.GET("/_cron/scheduler", basemw.Extend(gaemiddleware.RequireCron), func(c *router.Context) {
			cron.ScheduleReviews(c)
		})

		srv.Routes.GET("/_cron/update-config", basemw.Extend(gaemiddleware.RequireCron), func(c *router.Context) {
			cron.UpdateConfig(c)
		})

		return nil
	})
}
