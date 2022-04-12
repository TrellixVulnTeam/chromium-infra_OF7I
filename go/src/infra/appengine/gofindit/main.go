// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// package main implements the App Engine based HTTP server to handle request
// to GoFindit
package main

import (
	"context"
	"fmt"
	"infra/appengine/gofindit/compilefailureanalysis"
	"infra/appengine/gofindit/model"
	gfipb "infra/appengine/gofindit/proto"
	"infra/appengine/gofindit/pubsub"
	gfis "infra/appengine/gofindit/server"
	"net/http"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/auth/identity"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/encryptedcookies"
	"go.chromium.org/luci/server/templates"

	// Store auth sessions in the datastore.
	_ "go.chromium.org/luci/server/encryptedcookies/session/datastore"

	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/secrets"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	ACCESS_GROUP = "gofindit-access"
)

func init() {
	// TODO (crbug.com/1242998): Remove when this becomes the default (~Jan 2022).
	datastore.EnableSafeGet()
}

// checkAccess is middleware that checks if the user is authorized to
// access GoFindit.
func checkAccess(ctx *router.Context, next router.Handler) {
	user := auth.CurrentIdentity(ctx.Context)
	// User is not logged in
	if user.Kind() == identity.Anonymous {
		url, err := auth.LoginURL(ctx.Context, ctx.Request.URL.RequestURI())
		if err != nil {
			logging.Errorf(ctx.Context, "error in getting loginURL: %w", err)
			http.Error(ctx.Writer, "Error in getting loginURL", http.StatusInternalServerError)
		} else {
			http.Redirect(ctx.Writer, ctx.Request, url, http.StatusFound)
		}
		return
	}

	// User is logged in, check access group
	switch yes, err := auth.IsMember(ctx.Context, ACCESS_GROUP); {
	case err != nil:
		logging.Errorf(ctx.Context, "error in checking membership %s", err.Error())
		http.Error(ctx.Writer, "Error in authorizing the user.", http.StatusInternalServerError)
	case !yes:
		ctx.Writer.WriteHeader(http.StatusForbidden)
		templates.MustRender(ctx.Context, ctx.Writer, "pages/access-denied.html", nil)
	default:
		next(ctx)
	}
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
				"User":      auth.CurrentUser(ctx).Email,
				"LogoutURL": logoutURL,
			}, nil
		},
	}
}

func pageMiddlewareChain(srv *server.Server) router.MiddlewareChain {
	return router.NewMiddlewareChain(
		auth.Authenticate(srv.CookieAuth),
		templates.WithTemplates(prepareTemplates(&srv.Options)),
		checkAccess,
	)
}

func checkAPIAccess(ctx context.Context, methodName string, req proto.Message) (context.Context, error) {
	switch yes, err := auth.IsMember(ctx, ACCESS_GROUP); {
	case err != nil:
		return nil, status.Errorf(codes.Internal, "error when checking group membership")
	case !yes:
		return nil, status.Errorf(codes.PermissionDenied, "%s does not have access to method %s of GoFindit", auth.CurrentIdentity(ctx), methodName)
	default:
		return ctx, nil
	}
}

func main() {
	modules := []module.Module{
		gaeemulation.NewModuleFromFlags(),
		encryptedcookies.NewModuleFromFlags(), // Required for auth sessions.
		secrets.NewModuleFromFlags(),          // Needed by encryptedcookies.
	}

	server.Main(nil, modules, func(srv *server.Server) error {
		mwc := pageMiddlewareChain(srv)

		srv.Routes.GET("/", mwc, func(c *router.Context) {
			text := fmt.Sprintf("You are logged in as %s", auth.CurrentUser(c.Context).Identity.Email())
			c.Writer.Write([]byte(text))
		})

		// Pubsub handler
		// TODO (nqmtuan): Add auth for pubsub calls
		srv.Routes.POST("/_ah/push-handlers/buildbucket", nil, pubsub.BuildbucketPubSubHandler)

		// Installs PRPC service.
		gfipb.RegisterGoFinditServiceServer(srv.PRPC, &gfipb.DecoratedGoFinditService{
			Service: &gfis.GoFinditServer{},
			Prelude: checkAPIAccess,
		})

		srv.Routes.GET("/test", mwc, func(c *router.Context) {
			// For testing the flow
			// TODO (nqmtuan) remove this endpoint later
			failed_build := &model.LuciFailedBuild{
				Id: 88128398584903,
				LuciBuild: model.LuciBuild{
					BuildId:     88128398584903,
					Project:     "chromium",
					Bucket:      "ci",
					Builder:     "android",
					BuildNumber: 123,
					StartTime:   clock.Now(c.Context),
					EndTime:     clock.Now(c.Context),
					CreateTime:  clock.Now(c.Context),
				},
				FailureType: model.BuildFailureType_Compile,
			}
			if e := datastore.Put(c.Context, failed_build); e != nil {
				logging.Errorf(c.Context, "Got error when saving LuciFailedBuild entity: %v", e)
				return
			}

			compile_failure := &model.CompileFailure{
				Build:         datastore.KeyForObj(c.Context, failed_build),
				OutputTargets: []string{"abc.xyx"},
				Rule:          "CXX",
				Dependencies:  []string{"dep"},
			}
			if e := datastore.Put(c.Context, compile_failure); e != nil {
				logging.Errorf(c.Context, "Got error when saving CompileFailure entity: %v", e)
				return
			}

			_, e := compilefailureanalysis.AnalyzeFailure(c.Context, compile_failure, 8821136825293440641, 8821137635157166305)
			if e != nil {
				logging.Errorf(c.Context, "Got error when analyse failure: %v", e)
				return
			}
			c.Writer.Write([]byte("Testing"))
		})

		return nil
	})
}
