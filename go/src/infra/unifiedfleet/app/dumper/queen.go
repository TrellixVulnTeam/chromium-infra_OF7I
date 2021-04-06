// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dumper

import (
	"context"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
	"golang.org/x/oauth2"

	dronequeenapi "infra/appengine/drone-queen/api"
	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/util"
)

// pushToDroneQueen push the ufs duts to drone queen
func pushToDroneQueen(ctx context.Context) (err error) {
	// UFS migration done, run this job.
	if config.Get(ctx).GetEnableDronequeenPush() {
		defer func() {
			dumpPushToDroneQueenTick.Add(ctx, 1, err == nil)
		}()
		logging.Infof(ctx, "UFS migration done: pushToDroneQueen")
		client, err := getDroneQueenClient(ctx)
		if err != nil {
			return err
		}
		// Set namespace to OS to get only MachineLSEs for chromeOS.
		ctx, err = util.SetupDatastoreNamespace(ctx, util.OSNamespace)
		if err != nil {
			return err
		}
		// Get all the MachineLSEs
		// Set keysOnly to true to get only keys. This is faster and consumes less data.
		lses, err := inventory.ListAllMachineLSEs(ctx, true)
		if err != nil {
			err = errors.Annotate(err, "failed to list all MachineLSEs for chrome OS namespace").Err()
			logging.Errorf(ctx, err.Error())
			return err
		}
		sUnits, err := getAllSchedulingUnits(ctx, false)
		if err != nil {
			return err
		}
		var availableDuts []*dronequeenapi.DeclareDutsRequest_Dut
		// Map for MachineLSEs associated with SchedulingUnit for easy search.
		lseInSUnitMap := make(map[string]bool)
		for _, su := range sUnits {
			if len(su.GetMachineLSEs()) > 0 {
				availableDuts = append(availableDuts, &dronequeenapi.DeclareDutsRequest_Dut{
					Name: su.GetName(),
				})
				for _, lseName := range su.GetMachineLSEs() {
					lseInSUnitMap[lseName] = true
				}
			}
		}
		for _, lse := range lses {
			if !lseInSUnitMap[lse.GetName()] {
				availableDuts = append(availableDuts, &dronequeenapi.DeclareDutsRequest_Dut{
					Name: lse.GetName(),
					Hive: util.GetHiveForDut(lse.GetName()),
				})
			}
		}
		logging.Debugf(ctx, "DUTs to declare(%d): %+v", len(availableDuts), availableDuts)
		_, err = client.DeclareDuts(ctx, &dronequeenapi.DeclareDutsRequest{AvailableDuts: availableDuts})
		return err
	}
	logging.Infof(ctx, "UFS migration NOT done: skipping pushToDroneQueen")
	return nil
}

// getDroneQueenClient returns the drone queen client
func getDroneQueenClient(ctx context.Context) (dronequeenapi.InventoryProviderClient, error) {
	queenHostname := config.Get(ctx).QueenService
	if queenHostname == "" {
		logging.Errorf(ctx, "no drone queen service configured")
		return nil, errors.New("no drone queen service configured")
	}
	ts, err := auth.GetTokenSource(ctx, auth.AsSelf)
	if err != nil {
		return nil, err
	}
	h := oauth2.NewClient(ctx, ts)
	return dronequeenapi.NewInventoryProviderPRPCClient(&prpc.Client{
		C:    h,
		Host: queenHostname,
	}), nil
}

func getAllSchedulingUnits(ctx context.Context, keysOnly bool) ([]*ufspb.SchedulingUnit, error) {
	var sUnits []*ufspb.SchedulingUnit
	for startToken := ""; ; {
		res, nextToken, err := inventory.ListSchedulingUnits(ctx, pageSize, startToken, nil, keysOnly)
		if err != nil {
			return nil, errors.Annotate(err, "get all SchedulingUnits").Err()
		}
		sUnits = append(sUnits, res...)
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return sUnits, nil
}
