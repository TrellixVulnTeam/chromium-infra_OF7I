// Copyright 2018 The LUCI Authors.
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

// Package cron implements handlers for appengine cron targets in this app.
//
// All actual logic related to fleet management should be implemented in the
// main fleet API. These handlers should only encapsulate the following bits of
// logic:
// - Calling other API as the appengine service account user.
// - Translating luci-config driven admin task parameters.
package cron

import (
	"context"
	"net/http"

	"go.chromium.org/luci/appengine/gaemiddleware"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/proto/gerrit"
	"go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/server/router"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/app/clients"
	"infra/appengine/crosskylabadmin/app/config"
	"infra/appengine/crosskylabadmin/app/frontend"
	"infra/appengine/crosskylabadmin/app/frontend/inventory"
)

// InstallHandlers installs handlers for cron jobs that are part of this app.
//
// All handlers serve paths under /internal/cron/*
// These handlers can only be called by appengine's cron service.
func InstallHandlers(r *router.Router, mwBase router.MiddlewareChain) {
	mwCron := mwBase.Extend(gaemiddleware.RequireCron)
	r.GET("/internal/cron/refresh-inventory", mwCron, logAndSetHTTPErr(refreshInventoryCronHandler))
	r.GET("/internal/cron/balance-pools", mwCron, logAndSetHTTPErr(balancePoolCronHandler))

	// Generate repair or reset jobs for CrOS DUTs.
	r.GET("/internal/cron/push-bots-for-admin-tasks", mwCron, logAndSetHTTPErr(pushBotsForAdminTasksCronHandler))

	// for repair jobs of labstation.
	r.GET("/internal/cron/push-repair-jobs-for-labstations", mwCron, logAndSetHTTPErr(pushRepairJobsForLabstationsCronHandler))

	// Update device config asynchronously.
	r.GET("/internal/cron/update-device-configs", mwCron, logAndSetHTTPErr(updateDeviceConfigCronHandler))

	// Report Bot metrics.
	r.GET("/internal/cron/report-bots", mwCron, logAndSetHTTPErr(reportBotsCronHandler))
	// Report inventory metrics
	r.GET("/internal/cron/report-inventory", mwCron, logAndSetHTTPErr(reportInventoryCronHandler))

	r.GET("/internal/cron/push-inventory-to-queen", mwCron, logAndSetHTTPErr(pushInventoryToQueenCronHandler))

	// dump information from stable version file to datastore
	r.GET("/internal/cron/dump-stable-version-to-datastore", mwCron, logAndSetHTTPErr(dumpStableVersionToDatastoreHandler))
}

func updateDeviceConfigCronHandler(c *router.Context) (err error) {
	defer func() {
		updateDeviceConfigCronHandlerTick.Add(c.Context, 1, err == nil)
	}()
	inv := createInventoryServer(c)
	if _, err := inv.UpdateDeviceConfig(c.Context, &fleet.UpdateDeviceConfigRequest{}); err != nil {
		logging.Errorf(c.Context, "fail to update device config: %s", err.Error())
		return err
	}
	logging.Infof(c.Context, "update device config successfully")
	return nil
}

// pushBotsForAdminTasksCronHandler pushes bots that require admin tasks to bot queue.
func pushBotsForAdminTasksCronHandler(c *router.Context) (err error) {
	defer func() {
		pushBotsForAdminTasksCronHandlerTick.Add(c.Context, 1, err == nil)
	}()

	cfg := config.Get(c.Context)
	if cfg.RpcControl != nil && cfg.RpcControl.GetDisablePushBotsForAdminTasks() {
		logging.Infof(c.Context, "PushBotsForAdminTasks is disabled via config.")
		return nil
	}

	tsi := frontend.TrackerServerImpl{}
	if _, err := tsi.PushBotsForAdminTasks(c.Context, &fleet.PushBotsForAdminTasksRequest{}); err != nil {
		return err
	}
	logging.Infof(c.Context, "Successfully finished")
	return nil
}

// pushLabstationsForRepairCronHandler pushes bots that require admin tasks to bot queue.
func pushRepairJobsForLabstationsCronHandler(c *router.Context) (err error) {
	defer func() {
		pushRepairJobsForLabstationsCronHandlerTick.Add(c.Context, 1, err == nil)
	}()

	cfg := config.Get(c.Context)
	if cfg.RpcControl != nil && cfg.RpcControl.GetDisablePushLabstationsForRepair() {
		logging.Infof(c.Context, "PushLabstationsForRepair is disabled via config.")
		return nil
	}

	tsi := frontend.TrackerServerImpl{}
	if _, err := tsi.PushRepairJobsForLabstations(c.Context, &fleet.PushRepairJobsForLabstationsRequest{}); err != nil {
		return err
	}
	logging.Infof(c.Context, "Successfully finished")
	return nil
}

func reportBotsCronHandler(c *router.Context) (err error) {
	defer func() {
		reportBotsCronHandlerTick.Add(c.Context, 1, err == nil)
	}()

	tsi := frontend.TrackerServerImpl{}
	if _, err := tsi.ReportBots(c.Context, &fleet.ReportBotsRequest{}); err != nil {
		return err
	}
	logging.Infof(c.Context, "Successfully report bot metrics")
	return nil
}

func reportInventoryCronHandler(c *router.Context) (err error) {
	defer func() {
		reportInventoryCronHandlerTick.Add(c.Context, 1, err == nil)
	}()

	inv := createInventoryServer(c)
	_, err = inv.ReportInventory(c.Context, &fleet.ReportInventoryRequest{})
	return err
}

func pushInventoryToQueenCronHandler(c *router.Context) (err error) {
	defer func() {
		pushInventoryToQueenTick.Add(c.Context, 1, err == nil)
	}()
	inv := createInventoryServer(c)
	_, err = inv.PushInventoryToQueen(c.Context, &fleet.PushInventoryToQueenRequest{})
	return err
}

func logAndSetHTTPErr(f func(c *router.Context) error) func(*router.Context) {
	return func(c *router.Context) {
		if err := f(c); err != nil {
			http.Error(c.Writer, "Internal server error", http.StatusInternalServerError)
		}
	}
}

func refreshInventoryCronHandler(c *router.Context) error {
	cfg := config.Get(c.Context)
	if cfg.RpcControl != nil && cfg.RpcControl.GetDisableRefreshInventory() {
		return nil
	}
	inv := createInventoryServer(c)
	_, err := inv.UpdateCachedInventory(c.Context, &fleet.UpdateCachedInventoryRequest{})
	return err
}

func balancePoolCronHandler(c *router.Context) (err error) {
	cronCfg := config.Get(c.Context)
	if cronCfg.RpcControl.DisableEnsureCriticalPoolsHealthy {
		logging.Infof(c.Context, "EnsureCriticalPoolsHealthy is disabled via config.")
		return nil
	}

	cfg := config.Get(c.Context).GetCron().GetPoolBalancer()
	if cfg == nil {
		return errors.New("invalid pool balancer configuration")
	}

	inv := createInventoryServer(c)
	merr := make(errors.MultiError, 0)
	for _, target := range cfg.GetTargetPools() {
		resp, err := inv.BalancePools(c.Context, &fleet.BalancePoolsRequest{
			TargetPool:       target,
			SparePool:        cfg.GetSparePool(),
			MaxUnhealthyDuts: cfg.GetMaxUnhealthyDuts(),
		})
		if err != nil {
			logging.Errorf(c.Context, "Error in balancing pool for %s: %s", target, err.Error())
			merr = append(merr, errors.Annotate(err, "ensure critical pools healthy for pool %s", target).Err())
			continue
		}
		logging.Infof(c.Context, "Successfully balanced pool for target pool %s. Inventory change: %s", target, resp.GetGeneratedChangeUrl())
	}
	return merr.First()
}

func createInventoryServer(c *router.Context) *inventory.ServerImpl {
	tracker := &frontend.TrackerServerImpl{}
	return &inventory.ServerImpl{
		GerritFactory: func(c context.Context, host string) (gerrit.GerritClient, error) {
			return clients.NewGerritClientAsSelf(c, host)
		},
		GitilesFactory: func(c context.Context, host string) (gitiles.GitilesClient, error) {
			return clients.NewGitilesClientAsSelf(c, host)
		},
		TrackerFactory: func() fleet.TrackerServer {
			return tracker
		},
	}
}

func dumpStableVersionToDatastoreHandler(c *router.Context) error {
	logging.Infof(c.Context, "begin dumpStableVersionToDatastoreHandler")
	cfg := config.Get(c.Context)
	if cfg.RpcControl != nil && cfg.RpcControl.GetDisableDumpStableVersionToDatastore() {
		if cfg.RpcControl == nil {
			logging.Infof(c.Context, "end dumpStableVersionToDatastoreHandler immediately because RpcControl is nil")
		} else {
			logging.Infof(c.Context, "end dumpStableVersionToDatastoreHandler immediately because task is disabled")
		}
		return nil
	}
	inv := createInventoryServer(c)
	_, err := inv.DumpStableVersionToDatastore(c.Context, &fleet.DumpStableVersionToDatastoreRequest{})
	if err != nil {
		logging.Infof(c.Context, "end dumpStableVersionToDatastoreHandler with err (%s)", err)
	} else {
		logging.Infof(c.Context, "end dumpStableVersionToDatastoreHandler successfully")
	}
	return err
}
