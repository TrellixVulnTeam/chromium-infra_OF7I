// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"net/http"

	"go.chromium.org/luci/auth/identity"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/config/server/cfgmodule"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/cron"
	"go.chromium.org/luci/server/encryptedcookies"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/secrets"
	"go.chromium.org/luci/server/templates"

	// Store auth sessions in the datastore.
	_ "go.chromium.org/luci/server/encryptedcookies/session/datastore"

	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/config"
)

// authGroup is the name of the LUCI Auth group that controls whether the user
// should have access to Weetbix.
const authGroup = "weetbix-access"

func init() {
	// TODO (crbug.com/1242998): Remove when this becomes the default (~Jan 2022).
	datastore.EnableSafeGet()
}

// prepareTemplates configures templates.Bundle used by all UI handlers.
func prepareTemplates(opts *server.Options) *templates.Bundle {
	return &templates.Bundle{
		Loader: templates.FileSystemLoader("templates"),
		// Controls whether templates are cached.
		DebugMode: func(context.Context) bool { return !opts.Prod },
		DefaultArgs: func(ctx context.Context, e *templates.Extra) (templates.Args, error) {
			logoutURL, err := auth.LogoutURL(ctx, e.Request.URL.RequestURI())
			if err != nil {
				return nil, err
			}

			return templates.Args{
				"AuthGroup":       authGroup,
				"AuthServiceHost": opts.AuthServiceHost,
				"User":            auth.CurrentUser(ctx).Email,
				"LogoutURL":       logoutURL,
			}, nil
		},
	}
}

// requireAuth is middleware that forces the user to login and checks the
// user is authorised to use Weetbix before handling any request.
// If the user is not authorised, a standard "access is denied" page is
// displayed that allows the user to logout and login again with new
// credentials.
func requireAuth(ctx *router.Context, next router.Handler) {
	user := auth.CurrentIdentity(ctx.Context)
	if user.Kind() == identity.Anonymous {
		// User is not logged in.
		url, err := auth.LoginURL(ctx.Context, ctx.Request.URL.RequestURI())
		if err != nil {
			logging.Errorf(ctx.Context, "Fetching LoginURL: %s", err.Error())
			http.Error(ctx.Writer, "Internal server error while fetching Login URL.", http.StatusInternalServerError)
		} else {
			http.Redirect(ctx.Writer, ctx.Request, url, http.StatusFound)
		}
		return
	}

	isAuthorised, err := auth.IsMember(ctx.Context, authGroup)
	switch {
	case err != nil:
		logging.Errorf(ctx.Context, "Checking Auth Membership: %s", err.Error())
		http.Error(ctx.Writer, "Internal server error while checking authorisation.", http.StatusInternalServerError)
	case !isAuthorised:
		ctx.Writer.WriteHeader(http.StatusForbidden)
		templates.MustRender(ctx.Context, ctx.Writer, "pages/access-denied.html", nil)
	default:
		next(ctx)
	}
}

func pageBase(srv *server.Server) router.MiddlewareChain {
	return router.NewMiddlewareChain(
		auth.Authenticate(srv.CookieAuth),
		templates.WithTemplates(prepareTemplates(&srv.Options)),
		requireAuth,
	)
}

func indexPage(ctx *router.Context) {
	// TODO(crbug.com/1243174): Replace as part of MVP development.
	cfg, err := config.Get(ctx.Context)
	if err != nil {
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return
	}
	cl, err := bugs.NewMonorailClient(ctx.Context, cfg.GetMonorailHostname())
	if err != nil {
		logging.Errorf(ctx.Context, "Getting Monorail client: %s", err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return
	}
	issue, err := cl.GetIssue(ctx.Context, "projects/chromium/issues/2")
	if err != nil {
		logging.Errorf(ctx.Context, "Getting Monorail issue: %s", err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return
	}
	templates.MustRender(ctx.Context, ctx.Writer, "pages/index.html", templates.Args{
		"IssueTitle": issue.GetSummary(),
	})
}

func main() {
	modules := []module.Module{
		cfgmodule.NewModuleFromFlags(),
		cron.NewModuleFromFlags(),
		encryptedcookies.NewModuleFromFlags(), // Required for auth sessions.
		gaeemulation.NewModuleFromFlags(),     // Needed by cfgmodule.
		secrets.NewModuleFromFlags(),          // Needed by encryptedcookies.
	}
	server.Main(nil, modules, func(srv *server.Server) error {
		mw := pageBase(srv)
		srv.Routes.GET("/", mw, indexPage)

		// GAE crons.
		cron.RegisterHandler("read-config", config.Update)

		return nil
	})
}
