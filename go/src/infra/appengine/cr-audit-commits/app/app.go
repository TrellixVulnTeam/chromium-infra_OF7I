// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"go.chromium.org/luci/appengine/gaemiddleware"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/templates"
)

func main() {
	modules := []module.Module{
		gaeemulation.NewModuleFromFlags(),
	}

	server.Main(nil, modules, func(srv *server.Server) error {
		basemw := router.NewMiddlewareChain()

		templatesmw := basemw.Extend(templates.WithTemplates(&templates.Bundle{
			Loader:  templates.FileSystemLoader("templates"),
			FuncMap: templateFuncs,
		}))

		srv.Routes.GET("/", templatesmw, func(c *router.Context) {
			index(c)
		})

		// TODO: Temporarily remove this route. Will move it back after adding
		// user authentication support.
		/*
			srv.Routes.GET("/view/status", templatesmw, func(c *router.Context) {
				Status(c)
			})
		*/

		srv.Routes.GET("/_task/auditor", basemw.Extend(gaemiddleware.RequireTaskQueue("default")), func(c *router.Context) {
			Auditor(c)
		})

		srv.Routes.GET("/_cron/scheduler", basemw.Extend(gaemiddleware.RequireCron), func(c *router.Context) {
			Scheduler(c)
		})

		srv.Routes.GET("/admin/smoketest", basemw, func(c *router.Context) {
			SmokeTest(c)
		})

		return nil
	})
}

// Handler for the index page.
func index(rc *router.Context) {
	templates.MustRender(rc.Context, rc.Writer, "pages/index.html", templates.Args{})
}
