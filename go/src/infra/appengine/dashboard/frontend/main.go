// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"html/template"
	"net/http"
	"time"

	"go.chromium.org/luci/auth/identity"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/encryptedcookies"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/secrets"
	"go.chromium.org/luci/server/templates"

	// Using datastore for user sessions.
	_ "go.chromium.org/luci/server/encryptedcookies/session/datastore"

	dashpb "infra/appengine/dashboard/api/dashboard"
)

const authGroup = "chopsdash-access"

// prepareTemplates configures templates.Bundle used by all UI handlers.
func prepareTemplates(opts *server.Options) *templates.Bundle {
	return &templates.Bundle{
		Loader:    templates.FileSystemLoader("templates"),
		DebugMode: func(context.Context) bool { return !opts.Prod },
		FuncMap: template.FuncMap{
			"fmtDate": func(date time.Time) string {
				return date.Format("1-2-2006")
			},
		},
		DefaultArgs: func(ctx context.Context, e *templates.Extra) (templates.Args, error) {
			loginURL, err := auth.LoginURL(ctx, e.Request.URL.RequestURI())
			if err != nil {
				return nil, err
			}
			logoutURL, err := auth.LogoutURL(ctx, e.Request.URL.RequestURI())
			if err != nil {
				return nil, err
			}

			isAnonymous := true
			isGoogler := false
			isTrooper := false
			if auth.CurrentIdentity(ctx) != identity.AnonymousIdentity {
				isAnonymous = false
				isGoogler, err = auth.IsMember(ctx, authGroup)
				if err != nil {
					return nil, err
				}
				isTrooper, err = auth.IsMember(ctx, announcementGroup)
				if err != nil {
					return nil, err
				}
			}
			return templates.Args{
				"IsAnonymous": isAnonymous,
				"IsGoogler":   isGoogler,
				"IsTrooper":   isTrooper,
				"User":        auth.CurrentUser(ctx).Email,
				"LoginURL":    loginURL,
				"LogoutURL":   logoutURL,
			}, nil
		},
	}

}

func pageBase(srv *server.Server) router.MiddlewareChain {
	mw := router.NewMiddlewareChain(
		auth.Authenticate(
			&auth.GoogleOAuth2Method{Scopes: []string{"https://www.googleapis.com/auth/userinfo.email"}},
			srv.CookieAuth,
		),
	)
	return mw.Extend(templates.WithTemplates(prepareTemplates(&srv.Options)))
}

func main() {

	modules := []module.Module{
		encryptedcookies.NewModuleFromFlags(),
		secrets.NewModuleFromFlags(),
		gaeemulation.NewModuleFromFlags(),
	}

	server.Main(nil, modules, func(srv *server.Server) error {
		// When running locally, serve static files ourself.
		if !srv.Options.Prod {
			srv.Routes.Static("/static", router.MiddlewareChain{}, http.Dir("./static"))
		}

		// Register prpc API servers.
		dashpb.RegisterChopsServiceStatusServer(srv.PRPC, &dashboardService{})
		dashpb.RegisterChopsAnnouncementsServer(srv.PRPC, &dashpb.DecoratedChopsAnnouncements{
			Service: &announcementsServiceImpl{},
			Prelude: announcementsPrelude,
		})
		srv.PRPC.AccessControl = func(c context.Context, origin string) prpc.AccessControlDecision {
			// AccessAllowWithoutCredentials sets access control headers
			// (so sites like gerrit can query for announcements) without sharing
			// credentials (so we can use cookies auth).
			return prpc.AccessAllowWithoutCredentials
		}
		srv.PRPC.Authenticator = &auth.Authenticator{
			Methods: []auth.Method{
				&auth.GoogleOAuth2Method{
					Scopes: []string{"https://www.googleapis.com/auth/userinfo.email"},
				},
				srv.CookieAuth,
			},
		}

		srv.Routes.GET("/", pageBase(srv), dashboard)

		return nil
	})
}

func dashboard(ctx *router.Context) {
	templates.MustRender(ctx.Context, ctx.Writer, "pages/home.html", nil)
}
