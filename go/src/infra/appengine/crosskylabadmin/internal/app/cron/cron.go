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
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/internal/app/clients"
	"infra/appengine/crosskylabadmin/internal/app/config"
	"infra/appengine/crosskylabadmin/internal/app/frontend"
	"infra/appengine/crosskylabadmin/internal/app/frontend/inventory"
	"infra/appengine/crosskylabadmin/internal/app/gitstore"
)

// InstallHandlers installs handlers for cron jobs that are part of this app.
//
// All handlers serve paths under /internal/cron/*
// These handlers can only be called by appengine's cron service.
func InstallHandlers(r *router.Router, mwBase router.MiddlewareChain) {
	mwCron := mwBase.Extend(gaemiddleware.RequireCron)

	// Import config.cfg from LUCI Config.
	r.GET("/internal/cron/import-service-config", mwCron, logAndSetHTTPErr(importServiceConfig))

	// Generate repair jobs for needs_repair CrOS DUTs.
	r.GET("/internal/cron/push-bots-for-admin-tasks", mwCron, logAndSetHTTPErr(pushBotsForAdminTasksHandler(fleet.DutState_NeedsRepair)))

	// Generate repair jobs for repair_failed CrOS DUTs.
	r.GET("/internal/cron/push-repair-failed-bots-for-admin-tasks-hourly", mwCron, logAndSetHTTPErr(pushBotsForAdminTasksHandler(fleet.DutState_RepairFailed)))

	// Generate repair jobs for needs_manual_repair CrOS DUTs.
	r.GET("/internal/cron/push-repair-failed-bots-for-admin-tasks-daily", mwCron, logAndSetHTTPErr(pushBotsForAdminTasksHandler(fleet.DutState_NeedsManualRepair)))

	// for repair jobs of labstation.
	r.GET("/internal/cron/push-repair-jobs-for-labstations", mwCron, logAndSetHTTPErr(pushRepairJobsForLabstationsCronHandler))

	// Generate audit-usbkey jobs for CrOS DUTs.
	r.GET("/internal/cron/push-admin-audit-usbkey-tasks-for-duts", mwCron, logAndSetHTTPErr(pushAdminAuditActionHandler(fleet.AuditTask_ServoUSBKey)))

	// Generate audit-storage jobs for CrOS DUTs.
	r.GET("/internal/cron/push-admin-audit-storage-tasks-for-duts", mwCron, logAndSetHTTPErr(pushAdminAuditActionHandler(fleet.AuditTask_DUTStorage)))

	// Generate audit-rpm jobs for CrOS DUTs.
	r.GET("/internal/cron/push-admin-audit-rpm-tasks-for-duts", mwCron, logAndSetHTTPErr(pushAdminAuditActionHandler(fleet.AuditTask_RPMConfig)))

	// dump information from stable version file to datastore
	r.GET("/internal/cron/dump-stable-version-to-datastore", mwCron, logAndSetHTTPErr(dumpStableVersionToDatastoreHandler))
}

func importServiceConfig(c *router.Context) error {
	return config.Import(c.Context)
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

// pushAdminAuditActionHandler high-order for pushAdminAuditJobsCronHandler.
func pushAdminAuditActionHandler(auditTask fleet.AuditTask) (err func(c *router.Context) error) {
	return func(c *router.Context) error {
		return pushAdminAuditJobsCronHandler(c, auditTask)
	}
}

// pushAdminAuditJobsCronHandler pushes bots that will run audit tasks to bot queue.
func pushAdminAuditJobsCronHandler(c *router.Context, auditTask fleet.AuditTask) (err error) {
	defer func() {
		pushAdminAuditTasksCronHandlerTick.Add(c.Context, 1, err == nil)
	}()

	cfg := config.Get(c.Context)
	if cfg.RpcControl != nil && cfg.RpcControl.GetDisablePushDutsForAdminAudit() {
		logging.Infof(c.Context, "PushDutsForAdminAudit is disabled via config.")
		return nil
	}

	tsi := frontend.TrackerServerImpl{}
	req := &fleet.PushBotsForAdminAuditTasksRequest{
		Task: auditTask,
	}
	if _, err := tsi.PushBotsForAdminAuditTasks(c.Context, req); err != nil {
		return err
	}
	logging.Infof(c.Context, "Successfully finished")
	return nil
}

func logAndSetHTTPErr(f func(c *router.Context) error) func(*router.Context) {
	return func(c *router.Context) {
		if err := f(c); err != nil {
			http.Error(c.Writer, "Internal server error", http.StatusInternalServerError)
		}
	}
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
