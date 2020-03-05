// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package cron implements handlers for appengine cron targets in this app.
package cron

import (
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/gae/service/info"
	"go.chromium.org/luci/appengine/gaemiddleware"
	"go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/router"
	"golang.org/x/oauth2"

	apibq "infra/appengine/cros/lab_inventory/api/bigquery"
	"infra/appengine/cros/lab_inventory/app/config"
	"infra/appengine/cros/lab_inventory/app/converter"
	"infra/appengine/cros/lab_inventory/app/migration"
	dronequeenapi "infra/appengine/drone-queen/api"
	bqlib "infra/libs/cros/lab_inventory/bq"
	"infra/libs/cros/lab_inventory/cfg2datastore"
	"infra/libs/cros/lab_inventory/changehistory"
	"infra/libs/cros/lab_inventory/datastore"
	"infra/libs/cros/lab_inventory/deviceconfig"
	"infra/libs/cros/lab_inventory/dronecfg"
	"infra/libs/cros/lab_inventory/manufacturingconfig"
)

// InstallHandlers installs handlers for cron jobs that are part of this app.
//
// All handlers serve paths under /internal/cron/*
// These handlers can only be called by appengine's cron service.
func InstallHandlers(r *router.Router, mwBase router.MiddlewareChain) {
	mwCron := mwBase.Extend(gaemiddleware.RequireCron)
	r.GET("/internal/cron/dump-to-bq", mwCron, logAndSetHTTPErr(dumpToBQCronHandler))

	r.GET("/internal/cron/dump-registered-assets-snapshot", mwCron, logAndSetHTTPErr(dumpRegisteredAssetsCronHandler))

	r.GET("/internal/cron/dump-inventory-snapshot", mwCron, logAndSetHTTPErr(dumpInventorySnapshot))

	r.GET("/internal/cron/sync-dev-config", mwCron, logAndSetHTTPErr(syncDevConfigHandler))

	r.GET("/internal/cron/sync-manufacturing-config", mwCron, logAndSetHTTPErr(syncManufacturingConfigHandler))

	r.GET("/internal/cron/changehistory-to-bq", mwCron, logAndSetHTTPErr(dumpChangeHistoryToBQCronHandler))

	r.GET("/internal/cron/push-to-drone-queen", mwCron, logAndSetHTTPErr(pushToDroneQueenCronHandler))

	r.GET("/internal/cron/report-inventory", mwCron, logAndSetHTTPErr(reportInventoryCronHandler))

	r.GET("/internal/cron/sync-device-list-to-drone-config", mwCron, logAndSetHTTPErr(syncDeviceListToDroneConfigHandler))

	r.GET("/internal/cron/compare-inventory", mwCron, logAndSetHTTPErr(compareInventoryCronHandler))
}

func dumpToBQCronHandler(c *router.Context) (err error) {
	logging.Infof(c.Context, "not implemented yet")
	return nil
}

func syncDevConfigHandler(c *router.Context) error {
	logging.Infof(c.Context, "Start syncing device_config repo")
	cfg := config.Get(c.Context).GetDeviceConfigSource()
	cli, err := cfg2datastore.NewGitilesClient(c.Context, cfg.GetHost())
	if err != nil {
		return err
	}
	project := cfg.GetProject()
	committish := cfg.GetCommittish()
	path := cfg.GetPath()
	return deviceconfig.UpdateDatastore(c.Context, cli, project, committish, path)
}

func syncManufacturingConfigHandler(c *router.Context) error {
	logging.Infof(c.Context, "Start syncing manufacturing_config repo")
	cfg := config.Get(c.Context).GetManufacturingConfigSource()
	cli, err := cfg2datastore.NewGitilesClient(c.Context, cfg.GetHost())
	if err != nil {
		return err
	}
	project := cfg.GetProject()
	committish := cfg.GetCommittish()
	path := cfg.GetPath()
	return manufacturingconfig.UpdateDatastore(c.Context, cli, project, committish, path)
}

func dumpRegisteredAssetsCronHandler(c *router.Context) error {
	ctx := c.Context
	logging.Infof(ctx, "Start to dump registered assets to bigquery")

	uploader, err := bqlib.InitBQUploader(ctx, info.AppID(ctx), "inventory", fmt.Sprintf("registered_assets$%s", bqlib.GetPSTTimeStamp(time.Now())))
	if err != nil {
		return err
	}
	msgs := bqlib.GetRegisteredAssetsProtos(ctx)
	logging.Debugf(ctx, "Dumping %d records to bigquery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Dump is successfully finished")
	return nil
}

func dumpChangeHistoryToBQCronHandler(c *router.Context) error {
	ctx := c.Context
	logging.Infof(c.Context, "Start to dump change history to bigquery")
	project := info.AppID(ctx)
	dataset := "inventory"
	table := "changehistory"

	client, err := bigquery.NewClient(ctx, project)
	if err != nil {
		return err
	}
	up := bq.NewUploader(ctx, client, dataset, table)
	up.SkipInvalidRows = true
	up.IgnoreUnknownValues = true

	changes, err := changehistory.LoadFromDatastore(ctx)
	if err != nil {
		return err
	}
	msgs := make([]proto.Message, len(changes))
	for i, c := range changes {
		updatedTime, _ := ptypes.TimestampProto(c.Updated)
		msgs[i] = &apibq.ChangeHistory{
			Id:          c.DeviceID,
			Hostname:    c.Hostname,
			Label:       c.Label,
			OldValue:    c.OldValue,
			NewValue:    c.NewValue,
			UpdatedTime: updatedTime,
			ByWhom: &apibq.ChangeHistory_User{
				Name:  c.ByWhomName,
				Email: c.ByWhomEmail,
			},
			Comment: c.Comment,
		}
	}

	logging.Debugf(ctx, "Uploading %d records of change history", len(msgs))
	if err := up.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Cleaning %d records of change history from datastore", len(msgs))
	return changehistory.FlushDatastore(ctx, changes)
}

// dumpInventorySnapshot takes a snapshot of the inventory at the current time and
// uploads it to bigquery.
func dumpInventorySnapshot(c *router.Context) error {
	ctx := c.Context
	logging.Infof(c.Context, "Start dumping inventory snapshot")
	project := info.AppID(ctx)
	dataset := "inventory"
	table := "lab"
	client, err := bigquery.NewClient(ctx, project)
	if err != nil {
		return fmt.Errorf("bq client: %s", err)
	}

	logging.Debugf(ctx, "getting all devices")
	allDevices, err := datastore.GetAllDevices(ctx)
	logging.Debugf(ctx, "got devices (%d)", len(allDevices))
	if err != nil {
		return fmt.Errorf("gathering devices: %s", err)
	}

	labInventoryItems, err := converter.ToBQLabInventorySeq(allDevices)
	if err != nil {
		return fmt.Errorf("failed to convert devices: %s", err)
	}

	msgs := make([]proto.Message, len(labInventoryItems))

	for i, li := range labInventoryItems {
		msgs[i] = li
	}

	logging.Debugf(ctx, "uploading to bigquery dataset (%s) table (%s)", dataset, table)
	tz, _ := time.LoadLocation("America/Los_Angeles")
	up := bq.NewUploader(ctx, client, dataset, fmt.Sprintf("%s$%s", table, time.Now().In(tz).Format("20060102")))
	if err := up.Put(ctx, msgs...); err != nil {
		return fmt.Errorf("snapshot put: %s", err)
	}
	logging.Debugf(ctx, "successfully uploaded to bigquery")
	return nil
}

func pushToDroneQueenCronHandler(c *router.Context) error {
	ctx := c.Context
	logging.Infof(c.Context, "Start to push inventory to drone queen")
	queenHostname := config.Get(ctx).QueenService
	if queenHostname == "" {
		logging.Infof(ctx, "No drone queen service configured.")
		return nil
	}

	droneQueenRecord, err := dronecfg.Get(ctx, dronecfg.QueenDroneName(config.Get(ctx).Environment))
	if err != nil {
		return err
	}

	duts := make([]string, len(droneQueenRecord.DUTs))
	for i := range duts {
		duts[i] = droneQueenRecord.DUTs[i].Hostname
	}
	ts, err := auth.GetTokenSource(ctx, auth.AsSelf)
	if err != nil {
		return err
	}
	h := oauth2.NewClient(ctx, ts)
	client := dronequeenapi.NewInventoryProviderPRPCClient(&prpc.Client{
		C:    h,
		Host: queenHostname,
	})
	logging.Debugf(ctx, "DUTs to declare: %#v", duts)
	_, err = client.DeclareDuts(ctx, &dronequeenapi.DeclareDutsRequest{Duts: duts})
	if err != nil {
		return err
	}
	return nil
}

func reportInventoryCronHandler(c *router.Context) error {
	logging.Infof(c.Context, "start reporting inventory")
	if config.Get(c.Context).EnableInventoryReporting {
		return datastore.ReportInventory(c.Context, config.Get(c.Context).Environment)
	}
	logging.Infof(c.Context, "not enabled yet")
	return nil
}

func compareInventoryCronHandler(c *router.Context) error {
	logging.Infof(c.Context, "start to comparing inventory from v1 and v2")
	return migration.CompareInventory(c.Context)
}

func syncDeviceListToDroneConfigHandler(c *router.Context) error {
	ctx := c.Context
	logging.Infof(c.Context, "start to sync device list to drone config")
	return dronecfg.SyncDeviceList(ctx, dronecfg.QueenDroneName(config.Get(ctx).Environment))
}

func logAndSetHTTPErr(f func(c *router.Context) error) func(*router.Context) {
	return func(c *router.Context) {
		if err := f(c); err != nil {
			logging.Errorf(c.Context, err.Error())
			http.Error(c.Writer, "Internal server error", http.StatusInternalServerError)
		}
	}
}
