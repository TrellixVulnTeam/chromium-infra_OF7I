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
	"fmt"
	"net/http"
	"strings"

	"go.chromium.org/luci/appengine/gaemiddleware"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/app/clients"
	"infra/appengine/crosskylabadmin/app/config"
	"infra/appengine/crosskylabadmin/app/frontend"
	"infra/appengine/crosskylabadmin/app/frontend/inventory"
	"infra/appengine/crosskylabadmin/app/gitstore"
)

// InstallHandlers installs handlers for cron jobs that are part of this app.
//
// All handlers serve paths under /internal/cron/*
// These handlers can only be called by appengine's cron service.
func InstallHandlers(r *router.Router, mwBase router.MiddlewareChain) {
	mwCron := mwBase.Extend(gaemiddleware.RequireCron)

	// Import config.cfg from LUCI Config.
	r.GET("/internal/cron/import-service-config", mwCron, logAndSetHTTPErr(importServiceConfig))

	r.GET("/internal/cron/refresh-inventory", mwCron, logAndSetHTTPErr(refreshInventoryCronHandler))
	r.GET("/internal/cron/balance-pools", mwCron, logAndSetHTTPErr(balancePoolCronHandler))

	// Generate repair or reset jobs for CrOS DUTs.
	r.GET("/internal/cron/push-bots-for-admin-tasks", mwCron, logAndSetHTTPErr(pushBotsForAdminTasksHandler(fleet.DutState_NeedsRepair, fleet.DutState_NeedsReset)))

	// Generate repair or reset jobs for repair_failed CrOS DUTs.
	r.GET("/internal/cron/push-repair-failed-bots-for-admin-tasks-hourly", mwCron, logAndSetHTTPErr(pushBotsForAdminTasksHandler(fleet.DutState_RepairFailed)))

	// Generate repair or reset jobs for needs_manual_repair CrOS DUTs.
	r.GET("/internal/cron/push-repair-failed-bots-for-admin-tasks-daily", mwCron, logAndSetHTTPErr(pushBotsForAdminTasksHandler(fleet.DutState_NeedsManualRepair)))

	// for repair jobs of labstation.
	r.GET("/internal/cron/push-repair-jobs-for-labstations", mwCron, logAndSetHTTPErr(pushRepairJobsForLabstationsCronHandler))

	// Generate audit jobs for CrOS DUTs.
	r.GET("/internal/cron/push-admin-audit-tasks-for-duts", mwCron, logAndSetHTTPErr(pushAdminAuditJobsCronHandler))

	// Update device config asynchronously.
	r.GET("/internal/cron/update-device-configs", mwCron, logAndSetHTTPErr(updateDeviceConfigCronHandler))
	// Update (part of) manufacturing config asynchronously.
	r.GET("/internal/cron/update-manufacturing-configs", mwCron, logAndSetHTTPErr(updateManufacturingConfigCronHandler))

	// Report Bot metrics.
	r.GET("/internal/cron/report-bots", mwCron, logAndSetHTTPErr(reportBotsCronHandler))
	// Report inventory metrics
	r.GET("/internal/cron/report-inventory", mwCron, logAndSetHTTPErr(reportInventoryCronHandler))

	// dump information from stable version file to datastore
	r.GET("/internal/cron/dump-stable-version-to-datastore", mwCron, logAndSetHTTPErr(dumpStableVersionToDatastoreHandler))
}

func importServiceConfig(c *router.Context) error {
	return config.Import(c.Context)
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

func updateManufacturingConfigCronHandler(c *router.Context) (err error) {
	inv := createInventoryServer(c)
	if _, err := inv.UpdateManufacturingConfig(c.Context, &fleet.UpdateManufacturingConfigRequest{}); err != nil {
		logging.Errorf(c.Context, "fail to update manufacturing config: %s", err.Error())
		return err
	}
	logging.Infof(c.Context, "update manufacturing config successfully")
	return nil
}

// pushBotsForAdminTasksHandler high-order for pushBotsForAdminTasksCronHandler.
func pushBotsForAdminTasksHandler(dutStates ...fleet.DutState) (err func(c *router.Context) error) {
	return func(c *router.Context) error {
		return pushBotsForAdminTasksCronHandler(c, dutStates...)
	}
}

// pushBotsForAdminTasksCronHandler pushes bots that require admin tasks to bot queue.
func pushBotsForAdminTasksCronHandler(c *router.Context, dutStates ...fleet.DutState) (err error) {
	defer func() {
		pushBotsForAdminTasksCronHandlerTick.Add(c.Context, 1, err == nil)
	}()

	cfg := config.Get(c.Context)
	if cfg.RpcControl != nil && cfg.RpcControl.GetDisablePushBotsForAdminTasks() {
		logging.Infof(c.Context, "PushBotsForAdminTasks is disabled via config.")
		return nil
	}
	dutStateNames := make([]string, len(dutStates))
	for ind, dutState := range dutStates {
		dutStateNames[ind] = dutState.String()
	}
	logging.Infof(c.Context, fmt.Sprintf("PushBotsForAdminTasks for states: %v", strings.Join(dutStateNames, ", ")))

	tsi := frontend.TrackerServerImpl{}

	for _, dutState := range dutStates {
		logging.Infof(c.Context, fmt.Sprintf("Started push AdminTasks for state %#v", dutState.String()))
		request := fleet.PushBotsForAdminTasksRequest{
			TargetDutState: dutState,
		}
		if _, err := tsi.PushBotsForAdminTasks(c.Context, &request); err != nil {
			return err
		}
		logging.Infof(c.Context, fmt.Sprintf("Finished push AdminTasks for state %#v", dutState.String()))
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

// pushAdminAuditJobsCronHandler pushes bots that will run audit tasks to bot queue.
func pushAdminAuditJobsCronHandler(c *router.Context) (err error) {
	defer func() {
		pushAdminAuditTasksCronHandlerTick.Add(c.Context, 1, err == nil)
	}()

	cfg := config.Get(c.Context)
	if cfg.RpcControl != nil && cfg.RpcControl.GetDisablePushDutsForAdminAudit() {
		logging.Infof(c.Context, "PushDutsForAdminAudit is disabled via config.")
		return nil
	}

	tsi := frontend.TrackerServerImpl{}
	if _, err := tsi.PushBotsForAdminAuditTasks(c.Context, &fleet.PushBotsForAdminAuditTasksRequest{}); err != nil {
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
	cfg := config.Get(c.Context)
	_, err = inv.ReportInventory(c.Context, &fleet.ReportInventoryRequest{
		SkipInventoryMetrics: cfg.GetInventoryProvider().GetInventoryV2Only(),
	})
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
		GerritFactory: func(c context.Context, host string) (gitstore.GerritClient, error) {
			return clients.NewGerritClientAsSelf(c, host)
		},
		GitilesFactory: func(c context.Context, host string) (gitstore.GitilesClient, error) {
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
