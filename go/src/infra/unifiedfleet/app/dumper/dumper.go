// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dumper

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	bqlib "infra/cros/lab_inventory/bq"
	"infra/unifiedfleet/app/cron"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/util"
)

// Jobs is a list of all the cron jobs that are currently available for running
var Jobs = []*cron.CronTab{
	{
		// Dump configs, registrations, inventory and states to BQ
		Name:     util.CronJobNames["mainBQCron"],
		Time:     20 * time.Minute,
		TrigType: cron.DAILY,
		Job:      dump,
	},
	{
		// Dump change events to BQ
		Name:     util.CronJobNames["changeEventToBQCron"],
		Time:     10 * time.Minute,
		TrigType: cron.EVERY,
		Job:      dumpChangeEvent,
	},
	{
		// Dump snapshots to BQ
		Name:     util.CronJobNames["snapshotToBQCron"],
		Time:     10 * time.Minute,
		TrigType: cron.EVERY,
		Job:      dumpChangeSnapshots,
	},
	{
		// Dump network configs to BQ
		Name:     util.CronJobNames["networkConfigToBQCron"],
		Time:     60 * time.Minute,
		TrigType: cron.EVERY,
		Job:      dumpCrosNetwork,
	},
	{
		// Sync asset info from HaRT
		Name:     util.CronJobNames["hartSyncCron"],
		TrigType: cron.HOURLY,
		Job:      SyncAssetInfoFromHaRT,
	},
	{
		// Push changes to dron queen
		Name:     util.CronJobNames["droneQueenSyncCron"],
		Time:     10 * time.Minute,
		TrigType: cron.EVERY,
		Job:      pushToDroneQueen,
	},
	{
		// Report UFS metrics
		Name:     util.CronJobNames["InventoryMetricsReportCron"],
		Time:     5 * time.Minute,
		TrigType: cron.EVERY,
		Job:      reportUFSInventoryCronHandler,
	},
}

// InitServer initializes a cron server.
func InitServer(srv *server.Server) {
	for _, job := range Jobs {
		// make a copy of the job to avoid race condition.
		t := job
		// Start all the cron jobs in background.
		srv.RunInBackground(job.Name, func(ctx context.Context) {
			cron.Run(ctx, t)
		})
	}
}

// TriggerJob triggers a job by name. Returns error if the job is not found.
func TriggerJob(name string) error {
	for _, job := range Jobs {
		if job.Name == name {
			return cron.Trigger(job)
		}
	}
	return status.Errorf(codes.NotFound, "Invalid cron job %s. Not found", name)
}

func dump(ctx context.Context) error {
	ctx = logging.SetLevel(ctx, logging.Info)
	if err := exportToBQ(ctx, dumpToBQ); err != nil {
		return err
	}
	return nil
}

func dumpToBQ(ctx context.Context, bqClient *bigquery.Client) (err error) {
	defer func() {
		dumpToBQTick.Add(ctx, 1, err == nil)
	}()
	logging.Infof(ctx, "Dumping to BQ")
	curTime := time.Now()
	curTimeStr := bqlib.GetPSTTimeStamp(curTime)
	if err := configuration.SaveProjectConfig(ctx, &configuration.ProjectConfigEntity{
		Name:             getProject(ctx),
		DailyDumpTimeStr: curTimeStr,
	}); err != nil {
		return err
	}
	if err := dumpConfigurations(ctx, bqClient, curTimeStr); err != nil {
		return errors.Annotate(err, "dump configurations").Err()
	}
	if err := dumpRegistration(ctx, bqClient, curTimeStr); err != nil {
		return errors.Annotate(err, "dump registrations").Err()
	}
	if err := dumpInventory(ctx, bqClient, curTimeStr); err != nil {
		return errors.Annotate(err, "dump inventories").Err()
	}
	if err := dumpState(ctx, bqClient, curTimeStr); err != nil {
		return errors.Annotate(err, "dump states").Err()
	}
	logging.Debugf(ctx, "Dump is successfully finished")
	return nil
}

func dumpChangeEvent(ctx context.Context) (err error) {
	defer func() {
		dumpChangeEventTick.Add(ctx, 1, err == nil)
	}()
	ctx = logging.SetLevel(ctx, logging.Info)
	logging.Debugf(ctx, "Dumping change event to BQ")
	return exportToBQ(ctx, dumpChangeEventHelper)
}

func dumpChangeSnapshots(ctx context.Context) (err error) {
	defer func() {
		dumpChangeSnapshotTick.Add(ctx, 1, err == nil)
	}()
	ctx = logging.SetLevel(ctx, logging.Info)
	logging.Debugf(ctx, "Dumping change snapshots to BQ")
	return exportToBQ(ctx, dumpChangeSnapshotHelper)
}

func dumpCrosNetwork(ctx context.Context) (err error) {
	defer func() {
		dumpCrosNetworkTick.Add(ctx, 1, err == nil)
	}()
	// In UFS write to 'os' namespace
	ctx, err = util.SetupDatastoreNamespace(ctx, util.OSNamespace)
	if err != nil {
		return err
	}
	return importCrosNetwork(ctx)
}

// unique key used to store and retrieve context.
var contextKey = util.Key("ufs bigquery-client key")
var projectKey = util.Key("ufs project key")

// Use installs bigquery client to context.
func Use(ctx context.Context, bqClient *bigquery.Client) context.Context {
	return context.WithValue(ctx, contextKey, bqClient)
}

func get(ctx context.Context) *bigquery.Client {
	return ctx.Value(contextKey).(*bigquery.Client)
}

// UseProject installs project name to context.
func UseProject(ctx context.Context, project string) context.Context {
	return context.WithValue(ctx, projectKey, project)
}

func getProject(ctx context.Context) string {
	return ctx.Value(projectKey).(string)
}

func exportToBQ(ctx context.Context, f func(ctx context.Context, bqClient *bigquery.Client) error) error {
	var mErr error
	for _, ns := range util.ClientToDatastoreNamespace {
		newCtx, err := util.SetupDatastoreNamespace(ctx, ns)
		if ns == "" {
			// This is only for printing error message for default namespace.
			ns = "default (chrome)"
		}
		logging.Infof(newCtx, "Exporting to BQ for namespace %q", ns)
		if err != nil {
			logging.Errorf(ctx, "Setting namespace %q failed, BQ export skipped: %s", ns, err.Error())
			mErr = errors.NewMultiError(mErr, err)
			continue
		}
		err = f(newCtx, get(newCtx))
		if err != nil {
			logging.Errorf(ctx, "BQ export failed for the namespace %q: %s", ns, err.Error())
			mErr = errors.NewMultiError(mErr, err)
		}
	}
	return mErr
}
