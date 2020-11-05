// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"go.chromium.org/luci/appengine/gaemiddleware"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/router"

	"infra/appengine/rubber-stamper/cron"
)

func main() {
	server.Main(nil, nil, func(srv *server.Server) error {
		basemw := router.NewMiddlewareChain()

		srv.Routes.GET("/hello-world", router.MiddlewareChain{}, func(c *router.Context) {
			logging.Debugf(c.Context, "Hello world")
			c.Writer.Write([]byte("Hello, world. This is Rubber-Stamper."))
		})

		srv.Routes.GET("/_cron/scheduler", basemw.Extend(gaemiddleware.RequireCron), func(c *router.Context) {
			cron.ScheduleReviews(c)
		})

		return nil
	})
}
