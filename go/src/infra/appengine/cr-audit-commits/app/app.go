// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"go.chromium.org/luci/appengine/gaemiddleware"
	"go.chromium.org/luci/config/server/cfgmodule"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/templates"

	"infra/appengine/cr-audit-commits/app/config"
	cloudtasksmodule "infra/libs/grpcclient/cloudtasks"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
)

// app struct contains clients for services the app depends on.
// This is a lo-fi form of dependency injection, similar to what
// a tool like https://github.com/google/wire would use. Assign
// fake implementations, doubles etc for testing.
type app struct {
	cloudTasksClient *cloudtasks.Client
	opts             *server.Options
}

func main() {
	cloudtasksCfg := cloudtasksmodule.NewOptionsFromFlags()

	modules := []module.Module{
		gaeemulation.NewModuleFromFlags(),
		cfgmodule.NewModuleFromFlags(),
	}

	server.Main(nil, modules, func(srv *server.Server) error {
		basemw := router.NewMiddlewareChain()
		configmw := basemw.Extend(config.Middleware)
		templatesmw := basemw.Extend(templates.WithTemplates(&templates.Bundle{
			Loader:  templates.FileSystemLoader("templates"),
			FuncMap: templateFuncs,
		}))

		tasksClient, err := cloudtasksCfg.NewClient(srv.Context)
		if err != nil {
			return err
		}

		appServer := &app{
			cloudTasksClient: tasksClient,
			opts:             &srv.Options,
		}

		srv.Routes.GET("/", templatesmw, func(c *router.Context) {
			index(c)
		})

		srv.Routes.GET("/view/status", templatesmw.Extend(config.Middleware), func(c *router.Context) {
			Status(c)
		})

		srv.Routes.GET("/_task/auditor", configmw.Extend(gaemiddleware.RequireTaskQueue("default")), func(c *router.Context) {
			Auditor(c)
		})

		srv.Routes.GET("/_cron/scheduler", configmw.Extend(gaemiddleware.RequireCron), func(c *router.Context) {
			appServer.Schedule(c)
		})

		srv.Routes.GET("/_cron/update-config", basemw.Extend(gaemiddleware.RequireCron), func(c *router.Context) {
			config.Update(c)
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
