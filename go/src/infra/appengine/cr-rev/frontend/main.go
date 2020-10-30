// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Frontend service handles home page, API and redirects. Static files are
// served by GAE directly (configured in app.yaml).
package main

import (
	"infra/appengine/cr-rev/frontend/api/v1"
	"infra/appengine/cr-rev/frontend/redirect"
	"net/http"

	"go.chromium.org/luci/config/server/cfgmodule"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/templates"
)

const templatePath = "templates"

// handleIndex serves homepage of cr-rev
func handleIndex(c *router.Context) {
	templates.MustRender(
		c.Context, c.Writer, "pages/index.html", templates.Args{})
}

// handlePublicGerritRedirect redirects user to a CL on chromium-review
func handlePublicGerritRedirect(c *router.Context) {
	path := c.Params.ByName("path")
	url := "https://chromium-review.googlesource.com/c" + path
	http.Redirect(
		c.Writer, c.Request, url, http.StatusPermanentRedirect)
}

// handleInternalGerritRedirect redirects user to a CL on
// chrome-internal-review.
func handleInternalGerritRedirect(c *router.Context) {
	path := c.Params.ByName("path")
	url := "https://chrome-internal-review.googlesource.com/c" + path
	http.Redirect(
		c.Writer, c.Request, url, http.StatusPermanentRedirect)
}

// handleRedirect redirects user base on redirect rules. This is a catch-all
// redirect handler (e.g. crrev.com/3, crrev.com/{commit hash}). To add more
// rules, look at redirect package.
func handleRedirect(redirectRules *redirect.Rules, c *router.Context) {
	url, _, err := redirectRules.FindRedirectURL(c.Context, c.Request.RequestURI)
	switch err {
	case nil:
		http.Redirect(
			c.Writer, c.Request, url, http.StatusPermanentRedirect)
	case redirect.ErrNoMatch:
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
		redirect := redirect.NewRules(redirect.NewGitilesRedirect())

		srv.Routes.Handle("GET", "/i/*path", mw, handleInternalGerritRedirect)
		srv.Routes.Handle("GET", "/c/*path", mw, handlePublicGerritRedirect)
		srv.Routes.GET("/", mw, handleIndex)

		apiV1 := srv.Routes.Subrouter("/_ah/api/crrev/v1")
		api.NewRESTServer(apiV1, api.NewServer(redirect))

		// NotFound is used as catch-all.
		srv.Routes.NotFound(mw, func(c *router.Context) {
			handleRedirect(redirect, c)
		})
		return nil
	})
}
