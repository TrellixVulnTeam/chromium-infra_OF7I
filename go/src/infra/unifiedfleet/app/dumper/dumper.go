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

	bqlib "infra/cros/lab_inventory/bq"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/cron"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/util"
)

// Options is the dumper server configuration.
type Options struct {
	// CronInterval setups the user-specific cron interval for data dumping
	CronInterval time.Duration
}

// InitServer initializes a purger server.
func InitServer(srv *server.Server, opts Options) {
	srv.RunInBackground("ufs.dumper", func(ctx context.Context) {
		minInterval := 20 * time.Minute
		if opts.CronInterval > 0 {
			minInterval = opts.CronInterval
		}
		run(ctx, minInterval)
	})
	srv.RunInBackground("ufs.cros_inventory.dump", func(ctx context.Context) {
		cron.Run(ctx, 60*time.Minute, cron.EVERY, dumpCrosInventory)
	})
	srv.RunInBackground("ufs.change_event.BqDump", func(ctx context.Context) {
		cron.Run(ctx, 10*time.Minute, cron.EVERY, dumpChangeEvent)
	})
	srv.RunInBackground("ufs.snapshot_msg.BqDump", func(ctx context.Context) {
		cron.Run(ctx, 10*time.Minute, cron.EVERY, dumpChangeSnapshots)
	})
	srv.RunInBackground("ufs.cros_network.dump", func(ctx context.Context) {
		cron.Run(ctx, 60*time.Minute, cron.EVERY, dumpCrosNetwork)
	})
	/*srv.RunInBackground("ufs.sync_machines.sync", func(ctx context.Context) {
		cron.Run(ctx, 60*time.Minute, SyncMachinesFromAssets)
	})
	srv.RunInBackground("ufs.sync_devices.sync", func(ctx context.Context) {
		cron.Run(ctx, 60*time.Minute, SyncAssetInfoFromHaRT)
	})*/
	srv.RunInBackground("ufs.sync_assets.sync", func(ctx context.Context) {
		cron.Run(ctx, 5*time.Minute, cron.HOURLY, SyncAssetsFromIV2)
	})
	srv.RunInBackground("ufs.push_to_drone_queen", func(ctx context.Context) {
		cron.Run(ctx, 10*time.Minute, cron.EVERY, pushToDroneQueen)
	})
	srv.RunInBackground("ufs.dump_to_invV2_devices", func(ctx context.Context) {
		cron.Run(ctx, 10*time.Minute, cron.DAILY, DumpToInventoryDeviceSnapshot)
	})
	srv.RunInBackground("ufs.dump_to_invV2_dutstates", func(ctx context.Context) {
		cron.Run(ctx, 15*time.Minute, cron.DAILY, DumpToInventoryDutStateSnapshot)
	})
	srv.RunInBackground("ufs.report-inventory", func(ctx context.Context) {
		cron.Run(ctx, 5*time.Minute, cron.EVERY, reportUFSInventoryCronHandler)
	})
}

func run(ctx context.Context, minInterval time.Duration) {
	cron.Run(ctx, minInterval, cron.DAILY, dump)
}

func dump(ctx context.Context) error {
	ctx = logging.SetLevel(ctx, logging.Info)
	// Execute importing before dumping
	err1 := importCrimson(ctx)
	err2 := exportToBQ(ctx, dumpToBQ)
	if err1 == nil && err2 == nil {
		return nil
	}
	return errors.NewMultiError(err1, err2)
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

func dumpCrosInventory(ctx context.Context) (err error) {
	defer func() {
		dumpCrosInventoryTick.Add(ctx, 1, err == nil)
	}()
	// In UFS write to 'os' namespace
	// The below codes also use the ctx with OS namespace to query Inventory V2 and
	// is able to get response. As namespaces are datastore concepts, grpc knows nothing
	// about them and they generally do not apply to all APIs.
	ctx, err = util.SetupDatastoreNamespace(ctx, util.OSNamespace)
	if err != nil {
		return err
	}
	ctx = logging.SetLevel(ctx, logging.Info)
	crosInventoryHost := config.Get(ctx).CrosInventoryHost
	if crosInventoryHost == "" {
		crosInventoryHost = "cros-lab-inventory.appspot.com"
	}

	logging.Infof(ctx, "Comparing inventory V2 with UFS before importing")
	if err := compareInventoryV2(ctx, crosInventoryHost); err != nil {
		logging.Warningf(ctx, "Fail to generate sync diff: %s", err.Error())
	}
	logging.Infof(ctx, "Finish exporting diff from Inventory V2 to UFS to Google Storage")

	// UFS migration done, skip the import.
	if config.Get(ctx).GetDisableInv2Sync() {
		logging.Infof(ctx, "UFS migration done, skipping the InvV2 to UFS MachineLSE/DutState sync")
		return nil
	}
	return importCrosInventory(ctx, crosInventoryHost)
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

func exportToBQ(ctx context.Context, f func(ctx context.Context, bqClient *bigquery.Client) error) (err error) {
	for _, ns := range util.ClientToDatastoreNamespace {
		newCtx, err1 := util.SetupDatastoreNamespace(ctx, ns)
		if ns == "" {
			// This is only for printing error message for default namespace.
			ns = "default"
		}
		logging.Debugf(newCtx, "Exporting to BQ for namespace %q", ns)
		if err1 != nil {
			err1 = errors.Annotate(err, "Setting namespace %q failed. BQ export skipped for the namespace %q", ns, ns).Err()
			logging.Errorf(ctx, err.Error())
			err = errors.NewMultiError(err, err1)
			continue
		}
		err1 = f(newCtx, get(newCtx))
		if err1 != nil {
			err1 = errors.Annotate(err, "BQ export failed for the namespace %q", ns).Err()
			logging.Errorf(ctx, err.Error())
			err = errors.NewMultiError(err, err1)
		}
	}
	return err
}
