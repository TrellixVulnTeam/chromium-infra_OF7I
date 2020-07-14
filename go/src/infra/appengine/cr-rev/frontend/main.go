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
	"go.chromium.org/luci/server/templates"
)

const templatePath = "templates"

func handleIndex(c *router.Context) {
	templates.MustRender(
		c.Context, c.Writer, "pages/index.html", templates.Args{})
}

func main() {
	mw := router.MiddlewareChain{}
	mw = mw.Extend(templates.WithTemplates(&templates.Bundle{
		Loader:          templates.FileSystemLoader(templatePath),
		DefaultTemplate: "base",
	}))

	server.Main(nil, []module.Module{}, func(srv *server.Server) error {
		srv.Routes.GET("/", mw, handleIndex)
		return nil
	})
}
