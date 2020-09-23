// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"

	"go.chromium.org/luci/appengine/gaemiddleware"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/config/server/cfgmodule"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/templates"

	"infra/appengine/cr-audit-commits/app/config"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	gax "github.com/googleapis/gax-go/v2"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
)

func main() {
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

		sched := &scheduler{
			createTask: func(ctx context.Context, req *taskspb.CreateTaskRequest, opts ...gax.CallOption) (*taskspb.Task, error) {
				// Create a new cloud task client
				client, err := cloudtasks.NewClient(ctx)
				if err != nil {
					logging.WithError(err).Errorf(ctx, "Could not create cloud task client due to %s", err.Error())
					return nil, err
				}
				defer client.Close()

				return client.CreateTask(ctx, req)
			},
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
			sched.Schedule(c)
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
