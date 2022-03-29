// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// package main implements the App Engine based HTTP server to handle request
// to GoFindit
package main

import (
	"infra/appengine/gofindit/compilefailureanalysis"
	"infra/appengine/gofindit/model"
	gfipb "infra/appengine/gofindit/proto"
	gfis "infra/appengine/gofindit/server"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	"go.chromium.org/luci/server/router"
)

func init() {
	// TODO (crbug.com/1242998): Remove when this becomes the default (~Jan 2022).
	datastore.EnableSafeGet()
}

func main() {
	modules := []module.Module{
		gaeemulation.NewModuleFromFlags(),
	}

	server.Main(nil, modules, func(srv *server.Server) error {
		srv.Routes.GET("/", router.MiddlewareChain{}, func(c *router.Context) {
			c.Writer.Write([]byte("Placeholder for GoFindit UI"))
		})

		// Installs PRPC service.
		gfipb.RegisterGoFinditServiceServer(srv.PRPC, &gfipb.DecoratedGoFinditService{
			// TODO(nqmtuan): Check for auth here.
			Service: &gfis.GoFinditServer{},
		})

		srv.Routes.GET("/test", router.MiddlewareChain{}, func(c *router.Context) {
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
