// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"net/http"

	"go.chromium.org/luci/auth/identity"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/config/server/cfgmodule"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/cron"
	"go.chromium.org/luci/server/encryptedcookies"
	_ "go.chromium.org/luci/server/encryptedcookies/session/datastore"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/secrets"
	spanmodule "go.chromium.org/luci/server/span"
	"go.chromium.org/luci/server/templates"
	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/app"
	"infra/appengine/weetbix/frontend/handlers"
	"infra/appengine/weetbix/internal/admin"
	adminpb "infra/appengine/weetbix/internal/admin/proto"
	"infra/appengine/weetbix/internal/analysis"
	"infra/appengine/weetbix/internal/analyzedtestvariants"
	"infra/appengine/weetbix/internal/clustering/reclustering/orchestrator"
	"infra/appengine/weetbix/internal/config"
	"infra/appengine/weetbix/internal/metrics"
	"infra/appengine/weetbix/internal/services/reclustering"
	"infra/appengine/weetbix/internal/services/resultcollector"
	"infra/appengine/weetbix/internal/services/resultingester"
	"infra/appengine/weetbix/internal/services/testvariantbqexporter"
	"infra/appengine/weetbix/internal/services/testvariantupdator"
	"infra/appengine/weetbix/internal/services/testverdictingester"
	weetbixpb "infra/appengine/weetbix/proto/v1"
	"infra/appengine/weetbix/rpc"
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

			config, err := config.Get(ctx)
			if err != nil {
				return nil, err
			}

			return templates.Args{
				"AuthGroup":        authGroup,
				"AuthServiceHost":  opts.AuthServiceHost,
				"MonorailHostname": config.MonorailHostname,
				"User":             auth.CurrentUser(ctx).Email,
				"LogoutURL":        logoutURL,
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

func main() {
	modules := []module.Module{
		cfgmodule.NewModuleFromFlags(),
		cron.NewModuleFromFlags(),
		encryptedcookies.NewModuleFromFlags(), // Required for auth sessions.
		gaeemulation.NewModuleFromFlags(),     // Needed by cfgmodule.
		secrets.NewModuleFromFlags(),          // Needed by encryptedcookies.
		spanmodule.NewModuleFromFlags(),
		tq.NewModuleFromFlags(),
	}
	server.Main(nil, modules, func(srv *server.Server) error {
		mw := pageBase(srv)

		handlers := handlers.NewHandlers(srv.Options.CloudProject, srv.Options.Prod)
		handlers.RegisterRoutes(srv.Routes, mw)
		srv.Routes.Static("/static/", mw, http.Dir("./ui/dist"))
		// Anything that is not found, serve app html and let the client side router handle it.
		srv.Routes.NotFound(mw, handlers.IndexPage)

		// Register pPRC servers.
		srv.PRPC.AccessControl = prpc.AllowOriginAll
		srv.PRPC.Authenticator = &auth.Authenticator{
			Methods: []auth.Method{
				&auth.GoogleOAuth2Method{
					Scopes: []string{"https://www.googleapis.com/auth/userinfo.email"},
				},
			},
		}
		// TODO(crbug/1082369): Remove this workaround once field masks can be decoded.
		srv.PRPC.HackFixFieldMasksForJSON = true

		ac, err := analysis.NewClient(srv.Context, srv.Options.CloudProject)
		if err != nil {
			return errors.Annotate(err, "creating analysis client").Err()
		}
		weetbixpb.RegisterClustersServer(srv.PRPC, rpc.NewClustersServer(ac))
		weetbixpb.RegisterRulesServer(srv.PRPC, rpc.NewRulesSever())
		weetbixpb.RegisterProjectsServer(srv.PRPC, rpc.NewProjectsServer())
		weetbixpb.RegisterInitDataGeneratorServer(srv.PRPC, rpc.NewInitDataGeneratorServer())
		adminpb.RegisterAdminServer(srv.PRPC, admin.CreateServer())

		// GAE crons.
		cron.RegisterHandler("read-config", config.Update)
		cron.RegisterHandler("update-analysis-and-bugs", handlers.UpdateAnalysisAndBugs)
		cron.RegisterHandler("export-test-variants", testvariantbqexporter.ScheduleTasks)
		cron.RegisterHandler("purge-test-variants", analyzedtestvariants.Purge)
		cron.RegisterHandler("reclustering", orchestrator.CronHandler)
		cron.RegisterHandler("global-metrics", metrics.GlobalMetrics)

		// Pub/Sub subscription endpoints.
		srv.Routes.POST("/_ah/push-handlers/buildbucket", nil, app.BuildbucketPubSubHandler)
		srv.Routes.POST("/_ah/push-handlers/cvrun", nil, app.CVRunPubSubHandler)

		// Register task queue tasks.
		if err := reclustering.RegisterTaskHandler(srv); err != nil {
			return errors.Annotate(err, "register reclustering").Err()
		}
		if err := resultingester.RegisterTaskHandler(srv); err != nil {
			return errors.Annotate(err, "register result ingester").Err()
		}
		resultcollector.RegisterTaskClass()
		testvariantbqexporter.RegisterTaskClass()
		testvariantupdator.RegisterTaskClass()
		testverdictingester.RegisterTaskClass()

		return nil
	})
}
