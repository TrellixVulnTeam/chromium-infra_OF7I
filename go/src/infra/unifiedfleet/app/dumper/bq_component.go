// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dumper

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	bqlib "infra/cros/lab_inventory/bq"
	ufspb "infra/unifiedfleet/api/v1/models"
	apibq "infra/unifiedfleet/api/v1/models/bigquery"
	"infra/unifiedfleet/app/controller"
	"infra/unifiedfleet/app/model/caching"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
)

type getAllFunc func(ctx context.Context) ([]proto.Message, error)

var registrationDumpToolkit = map[string]getAllFunc{
	"assets":   getAllAssetMsgs,
	"machines": getAllMachineMsgs,
	"racks":    getAllRackMsgs,
	"kvms":     getAllKVMMsgs,
	"switches": getAllSwitchMsgs,
	"rpms":     getAllRpmMsgs,
	"nics":     getAllNicMsgs,
	"dracs":    getAllDracMsgs,
}

var inventoryDumpToolkit = map[string]getAllFunc{
	"machine_lses":            getAllMachineLSEsMsgs,
	"rack_lses":               getAllRackLSEMsgs,
	"vms":                     getAllVMMsgs,
	"caching_services":        getAllCachingServiceMsgs,
	"machine_lse_deployments": getAllMachineLSEDeploymentsMsgs,
}

var stateDumpToolkit = map[string]getAllFunc{
	"state_records": getAllStateRecordMsgs,
	"dutstates":     getAllDutStateRecordMsgs,
}

var configurationDumpToolkit = map[string]getAllFunc{
	"chrome_platforms":       getAllChromePlatformMsgs,
	"vlans":                  getAllVlanMsgs,
	"rack_lse_prototypes":    getAllRackLSEPrototypeMsgs,
	"machine_lse_prototypes": getAllMachineLSEPrototypeMsgs,
	"dhcps":                  getAllDHCPMsgs,
	"ips":                    getAllIPMsgs,
}

func dumpHelper(ctx context.Context, bqClient *bigquery.Client, msgs []proto.Message, table, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("%s$%s", table, curTimeStr))
	logging.Infof(ctx, "Dumping %d %s records to BigQuery", len(msgs), table)
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Infof(ctx, "Finish dumping successfully")
	return nil
}

func getAllVMMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := controller.ListVMs(ctx, pageSize, startToken, "", false)
		if err != nil {
			return nil, errors.Annotate(err, "get all vlans").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.VMRow{
				Vm: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllChromePlatformMsgs(ctx context.Context) ([]proto.Message, error) {
	platforms, err := configuration.GetAllChromePlatforms(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "get all chrome platforms").Err()
	}
	msgs := make([]proto.Message, 0)
	for _, p := range *platforms {
		if p.Err != nil {
			continue
		}
		msg := &apibq.ChromePlatformRow{
			Platform: p.Data.(*ufspb.ChromePlatform),
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

func getAllVlanMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := controller.ListVlans(ctx, pageSize, startToken, "", false)
		if err != nil {
			return nil, errors.Annotate(err, "get all vlans").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.VlanRow{
				Vlan: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllRackLSEPrototypeMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := configuration.ListRackLSEPrototypes(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return nil, errors.Annotate(err, "get all rack lse prototypes").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.RackLSEPrototypeRow{
				RackLsePrototype: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllMachineLSEPrototypeMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := configuration.ListMachineLSEPrototypes(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return nil, errors.Annotate(err, "get all vlans").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.MachineLSEPrototypeRow{
				MachineLsePrototype: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllDHCPMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := configuration.ListDHCPConfigs(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return nil, errors.Annotate(err, "get all dhcps").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.DHCPConfigRow{
				DhcpConfig: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllIPMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := configuration.ListIPs(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return nil, errors.Annotate(err, "get all dhcps").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.IPRow{
				Ip: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllMachineMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := registration.ListMachines(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return nil, errors.Annotate(err, "get all machines").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.MachineRow{
				Machine: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllRackMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := registration.ListRacks(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return nil, errors.Annotate(err, "get all racks").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.RackRow{
				Rack: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllAssetMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := registration.ListAssets(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return nil, errors.Annotate(err, "get all assets").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.AssetRow{
				Asset: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllKVMMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := registration.ListKVMs(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return nil, errors.Annotate(err, "get all kvms").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.KVMRow{
				Kvm: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllSwitchMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := registration.ListSwitches(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return nil, errors.Annotate(err, "get all switches").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.SwitchRow{
				Switch: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllRpmMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := registration.ListRPMs(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return nil, errors.Annotate(err, "get all rpms").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.RPMRow{
				Rpm: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllNicMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := registration.ListNics(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return nil, errors.Annotate(err, "get all nics").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.NicRow{
				Nic: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllDracMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := registration.ListDracs(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return nil, errors.Annotate(err, "get all dracs").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.DracRow{
				Drac: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllMachineLSEsMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := inventory.ListMachineLSEs(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return nil, errors.Annotate(err, "get all machine lses").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.MachineLSERow{
				MachineLse: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllMachineLSEDeploymentsMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := inventory.ListMachineLSEDeployments(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return nil, errors.Annotate(err, "get all machine lse deployments").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.MachineLSEDeploymentRow{
				MachineLseDeployment: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllRackLSEMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := inventory.ListRackLSEs(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return nil, errors.Annotate(err, "get all rack lses").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.RackLSERow{
				RackLse: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllStateRecordMsgs(ctx context.Context) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := state.ListStateRecords(ctx, pageSize, startToken, nil)
		if err != nil {
			return nil, errors.Annotate(err, "get all state records").Err()
		}
		for _, r := range res {
			msgs = append(msgs, &apibq.StateRecordRow{
				StateRecord: r,
			})
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return msgs, nil
}

func getAllDutStateRecordMsgs(ctx context.Context) ([]proto.Message, error) {
	states, err := state.ListAllDutStates(ctx, false)
	if err != nil {
		return nil, err
	}
	msgs := make([]proto.Message, len(states))
	for idx, state := range states {
		msgs[idx] = &apibq.DUTStateRecordRow{
			State: state,
		}
	}
	return msgs, nil
}

func getAllCachingServiceMsgs(ctx context.Context) ([]proto.Message, error) {
	cachingServices, err := caching.ListAllCachingServices(ctx, false)
	if err != nil {
		return nil, errors.Annotate(err, "get all caching services").Err()
	}
	msgs := make([]proto.Message, 0)
	for _, p := range cachingServices {
		msg := &apibq.CachingServiceRow{
			CachingService: p,
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}
