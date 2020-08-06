// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Frontend service handles home page, API and redirects. Static files are
// served by GAE directly (configured in app.yaml).
package main

import (
	"net/http"

	"go.chromium.org/luci/config/server/cfgmodule"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/templates"
)

const templatePath = "templates"

func handleIndex(c *router.Context) {
	templates.MustRender(
		c.Context, c.Writer, "pages/index.html", templates.Args{})
}

func handleRedirect(redirect *redirectRules, c *router.Context) {
	url, err := redirect.findRedirectURL(c.Request.RequestURI)
	switch err {
	case nil:
		http.Redirect(
			c.Writer, c.Request, url, http.StatusPermanentRedirect)
	case errNoMatch:
		http.NotFound(c.Writer, c.Request)
	default:
		http.Error(
			c.Writer, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	mw := router.MiddlewareChain{}
	mw = mw.Extend(templates.WithTemplates(&templates.Bundle{
		Loader:          templates.FileSystemLoader(templatePath),
		DefaultTemplate: "base",
	}))

	modules := []module.Module{
		cfgmodule.NewModuleFromFlags(),
		gaeemulation.NewModuleFromFlags(),
	}

	server.Main(nil, modules, func(srv *server.Server) error {
		redirect := newRedirectRules()

		srv.Routes.GET("/", mw, handleIndex)
		// NotFound is used as catch-all.
		srv.Routes.NotFound(mw, func(c *router.Context) {
			handleRedirect(redirect, c)
		})
		return nil
	})
}
