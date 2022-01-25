// Copyright 2019 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package queue implements handlers for taskqueue jobs in this app.
//
// All actual logic are implemented in tasker layer.
package queue

import (
	"math/rand"
	"net/http"

	"go.chromium.org/luci/appengine/gaemiddleware"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"

	"infra/appengine/crosskylabadmin/internal/app/config"
	"infra/appengine/crosskylabadmin/internal/app/frontend"
	"infra/appengine/crosskylabadmin/internal/ufs"
)

// InstallHandlers installs handlers for queue jobs that are part of this app.
//
// All handlers serve paths under /internal/queue/*
func InstallHandlers(r *router.Router, mwBase router.MiddlewareChain) {
	r.POST(
		"/internal/task/cros_repair/*ignored",
		mwBase.Extend(gaemiddleware.RequireTaskQueue("repair-bots")),
		logAndSetHTTPErr(runRepairQueueHandler),
	)
	r.POST(
		"/internal/task/labstation_repair/*ignored",
		mwBase.Extend(gaemiddleware.RequireTaskQueue("repair-labstations")),
		logAndSetHTTPErr(runRepairQueueHandler),
	)
	r.POST(
		"/internal/task/audit/*ignored",
		mwBase.Extend(gaemiddleware.RequireTaskQueue("audit-bots")),
		logAndSetHTTPErr(runAuditQueueHandler),
	)
}

func runRepairQueueHandler(c *router.Context) (err error) {
	defer func() {
		runRepairTick.Add(c.Context, 1, err == nil)
	}()
	// Create a UFS client at the beginning of repair and log the result, but do NOT stop execution
	// because of problems. We are not yet ready to make UFS a hard dependency of CSA, so at this point
	// it is a soft dependency. The UFS client can be nil.
	//
	// We are going to use the pools associated with a device as an input to decide which implementation
	// of repair to use.
	cfg := config.Get(c.Context)
	hc, err := ufs.NewHTTPClient(c.Context)
	if err != nil {
		logging.Infof(c.Context, "run repair queue handler: %s", err)
	}
	ufsClient, err := ufs.NewClient(c.Context, hc, cfg.GetUFS().GetHost())
	if err == nil {
		logging.Infof(c.Context, "run repair queue handler: UFS client created successfully")
	} else {
		logging.Infof(c.Context, "run repair queue handler: %s", err)
	}
	botID := c.Request.FormValue("botID")
	expectedState := c.Request.FormValue("expectedState")
	// RandFloat is guaranteed to be in the half-open interval [0,1).
	randFloat := rand.Float64()
	pools, err := ufs.GetPools(c.Context, ufsClient, botID)
	// Failure to look up the pools associated with a device is non-fatal.
	// We will take a safe action inside CreateRepairTask. Log and move on.
	if err != nil {
		logging.Infof(c.Context, "run repair queue handler: %s", err)
	}
	taskURL, err := frontend.CreateRepairTask(c.Context, botID, expectedState, pools, randFloat)
	if err != nil {
		logging.Infof(c.Context, "fail to run repair job in queue for %s: %s", botID, err.Error())
		return err
	}

	logging.Infof(c.Context, "Successfully run repair job for %s: %s", botID, taskURL)
	return nil
}

func runAuditQueueHandler(c *router.Context) (err error) {
	defer func() {
		runAuditTick.Add(c.Context, 1, err == nil)
	}()

	botID := c.Request.FormValue("botID")
	actions := c.Request.FormValue("actions")
	taskname := c.Request.FormValue("taskname")
	taskURL, err := frontend.CreateAuditTask(c.Context, botID, taskname, actions)
	if err != nil {
		return err
	}
	logging.Infof(c.Context, "Successfully run audit job for %s: %s", botID, taskURL)
	return nil
}

func logAndSetHTTPErr(f func(c *router.Context) error) func(*router.Context) {
	return func(c *router.Context) {
		if err := f(c); err != nil {
			http.Error(c.Writer, "Internal server error", http.StatusInternalServerError)
		}
	}
}
