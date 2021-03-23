// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dumper

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"

	invV1 "infra/libs/skylab/inventory"
	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"
)

type inventoryCounter map[bucket]int

var inventoryMetric = metric.NewInt(
	"chromeos/skylab/inventory/dut_count",
	"The number of DUTs in a given bucket",
	nil,
	field.String("board"),
	field.String("model"),
	field.String("pool"),
	field.String("environment"),
)

// reportUFSInventoryCronHandler push the ufs duts metrics to tsmon
func reportUFSInventoryCronHandler(ctx context.Context) (err error) {
	logging.Infof(ctx, "Reporting UFS inventory DUT metrics")
	env := config.Get(ctx).SelfStorageBucket
	// Set namespace to OS to get only MachineLSEs for chromeOS.
	ctx, err = util.SetupDatastoreNamespace(ctx, util.OSNamespace)
	if err != nil {
		return err
	}
	// Get all the MachineLSEs
	lses, err := inventory.ListAllMachineLSEs(ctx, false)
	if err != nil {
		return err
	}
	machines, err := registration.ListAllMachines(ctx, false)
	if err != nil {
		return err
	}
	idTomachineMap := make(map[string]*ufspb.Machine, 0)
	for _, machine := range machines {
		idTomachineMap[machine.GetName()] = machine
	}
	c := make(inventoryCounter)
	for _, lse := range lses {
		if len(lse.GetMachines()) < 0 {
			continue
		}
		machine, ok := idTomachineMap[lse.GetMachines()[0]]
		if !ok {
			continue
		}
		b := getBucketForDevice(lse, machine, env)
		c[b]++
	}
	logging.Infof(ctx, "report UFS inventory metrics for %d devices", len(c))
	c.Report(ctx)
	return nil
}

func (c inventoryCounter) Report(ctx context.Context) {
	for b, count := range c {
		//logging.Infof(ctx, "bucket: %s, number: %d", b.String(), count)
		inventoryMetric.Set(ctx, int64(count), b.board, b.model, b.pool, b.environment)
	}
}

func getBucketForDevice(lse *ufspb.MachineLSE, machine *ufspb.Machine, env string) bucket {
	b := bucket{
		board:       machine.GetChromeosMachine().GetBuildTarget(),
		model:       machine.GetChromeosMachine().GetModel(),
		pool:        "[None]",
		environment: env,
	}
	if dut := lse.GetChromeosMachineLse().GetDeviceLse().GetDut(); dut != nil {
		b.pool = getReportPool(dut.GetPools())
	}
	if labstation := lse.GetChromeosMachineLse().GetDeviceLse().GetLabstation(); labstation != nil {
		b.pool = getReportPool(labstation.GetPools())
	}
	return b
}

// bucket contains static DUT dimensions.
//
// These dimensions do not change often. If all DUTs with a given set of
// dimensions are removed, the related metric is not automatically reset. The
// metric will get reset eventually.
type bucket struct {
	board       string
	model       string
	pool        string
	environment string
}

func (b bucket) String() string {
	return fmt.Sprintf("board: %s, model: %s, pool: %s, env: %s", b.board, b.model, b.pool, b.environment)
}

func summarizeValues(vs []string) string {
	switch len(vs) {
	case 0:
		return "[None]"
	case 1:
		return vs[0]
	default:
		return "[Multiple]"
	}
}

func isManagedPool(p string) bool {
	_, ok := invV1.SchedulableLabels_DUTPool_value[p]
	return ok
}

func getReportPool(pools []string) string {
	p := summarizeValues(pools)
	if isManagedPool(p) {
		return fmt.Sprintf("managed:%s", p)
	}
	return p
}
