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

	bqlib "infra/libs/cros/lab_inventory/bq"
	ufspb "infra/unifiedfleet/api/v1/proto"
	apibq "infra/unifiedfleet/api/v1/proto/bigquery"
	"infra/unifiedfleet/app/controller"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
)

const ufsDatasetName = "ufs"
const pageSize = 500

func dumpChangeEventHelper(ctx context.Context, bqClient *bigquery.Client) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, "change_events")
	changes, err := history.GetAllChangeEventEntities(ctx)
	if err != nil {
		return errors.Annotate(err, "get all change events' entities").Err()
	}
	msgs := make([]proto.Message, 0)
	for _, p := range changes {
		logging.Debugf(ctx, "%#v", p)
		data, err := p.GetProto()
		if err != nil {
			continue
		}
		msg := &apibq.ChangeEventRow{
			ChangeEvent: data.(*ufspb.ChangeEvent),
		}
		msgs = append(msgs, msg)
	}
	logging.Debugf(ctx, "Dumping %d change events to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		logging.Debugf(ctx, "fail to upload: %s", err.Error())
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	logging.Debugf(ctx, "Deleting uploaded entities")
	if err := history.DeleteChangeEventEntities(ctx, changes); err != nil {
		logging.Debugf(ctx, "fail to delete entities: %s", err.Error())
		return err
	}
	logging.Debugf(ctx, "Finish deleting successfully")
	return nil
}

func dumpConfigurations(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) (err error) {
	if err := dumpChromePlatforms(ctx, bqClient, curTimeStr); err != nil {
		return err
	}
	if err := dumpVlans(ctx, bqClient, curTimeStr); err != nil {
		return err
	}
	if err := dumpRackLSEPrototypes(ctx, bqClient, curTimeStr); err != nil {
		return err
	}
	if err := dumpMachineLSEPrototypes(ctx, bqClient, curTimeStr); err != nil {
		return err
	}
	if err := dumpDHCPs(ctx, bqClient, curTimeStr); err != nil {
		return err
	}
	if err := dumpIPs(ctx, bqClient, curTimeStr); err != nil {
		return err
	}
	return nil
}

func dumpRegistration(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) (err error) {
	if err := dumpMachines(ctx, bqClient, curTimeStr); err != nil {
		return err
	}
	if err := dumpRacks(ctx, bqClient, curTimeStr); err != nil {
		return err
	}
	if err := dumpKVMs(ctx, bqClient, curTimeStr); err != nil {
		return err
	}
	if err := dumpSwitches(ctx, bqClient, curTimeStr); err != nil {
		return err
	}
	if err := dumpRPMs(ctx, bqClient, curTimeStr); err != nil {
		return err
	}
	if err := dumpNics(ctx, bqClient, curTimeStr); err != nil {
		return err
	}
	if err := dumpDracs(ctx, bqClient, curTimeStr); err != nil {
		return err
	}
	return nil
}

func dumpInventory(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) (err error) {
	if err := dumpMachineLSEs(ctx, bqClient, curTimeStr); err != nil {
		return err
	}
	if err := dumpRackLSEs(ctx, bqClient, curTimeStr); err != nil {
		return err
	}
	return nil
}

func dumpState(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) (err error) {
	if err := dumpStateRecord(ctx, bqClient, curTimeStr); err != nil {
		return err
	}
	return nil
}

func dumpChromePlatforms(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("chrome_platforms$%s", curTimeStr))
	platforms, err := configuration.GetAllChromePlatforms(ctx)
	if err != nil {
		return errors.Annotate(err, "get all chrome platforms").Err()
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
	logging.Debugf(ctx, "Dumping %d chrome_platform records to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	return nil
}

func dumpVlans(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("vlans$%s", curTimeStr))
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := controller.ListVlans(ctx, pageSize, startToken, "", false)
		if err != nil {
			return errors.Annotate(err, "get all vlans").Err()
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
	logging.Debugf(ctx, "Dumping %d vlan records to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	return nil
}

func dumpRackLSEPrototypes(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("rack_lse_prototypes$%s", curTimeStr))
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := configuration.ListRackLSEPrototypes(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return errors.Annotate(err, "get all rack lse prototypes").Err()
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
	logging.Debugf(ctx, "Dumping %d rack lse prototypes records to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	return nil
}

func dumpMachineLSEPrototypes(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("machine_lse_prototypes$%s", curTimeStr))
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := configuration.ListMachineLSEPrototypes(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return errors.Annotate(err, "get all vlans").Err()
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
	logging.Debugf(ctx, "Dumping %d machine lse prototypes records to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	return nil
}

func dumpDHCPs(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("dhcps$%s", curTimeStr))
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := configuration.ListDHCPConfigs(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return errors.Annotate(err, "get all dhcps").Err()
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
	logging.Debugf(ctx, "Dumping %d dhcp records to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	return nil
}

func dumpIPs(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("ips$%s", curTimeStr))
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := configuration.ListIPs(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return errors.Annotate(err, "get all dhcps").Err()
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
	logging.Debugf(ctx, "Dumping %d ip records to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	return nil
}

func dumpMachines(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("machines$%s", curTimeStr))
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := registration.ListMachines(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return errors.Annotate(err, "get all machines").Err()
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
	logging.Debugf(ctx, "Dumping %d machine records to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	return nil
}

func dumpRacks(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("racks$%s", curTimeStr))
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := registration.ListRacks(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return errors.Annotate(err, "get all racks").Err()
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
	logging.Debugf(ctx, "Dumping %d rack records to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	return nil
}

func dumpKVMs(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("kvms$%s", curTimeStr))
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := registration.ListKVMs(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return errors.Annotate(err, "get all kvms").Err()
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
	logging.Debugf(ctx, "Dumping %d kvm records to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	return nil
}

func dumpSwitches(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("switches$%s", curTimeStr))
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := registration.ListSwitches(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return errors.Annotate(err, "get all switches").Err()
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
	logging.Debugf(ctx, "Dumping %d switch records to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	return nil
}

func dumpRPMs(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("rpms$%s", curTimeStr))
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := registration.ListRPMs(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return errors.Annotate(err, "get all rpms").Err()
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
	logging.Debugf(ctx, "Dumping %d rpm records to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	return nil
}

func dumpNics(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("nics$%s", curTimeStr))
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := registration.ListNics(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return errors.Annotate(err, "get all nics").Err()
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
	logging.Debugf(ctx, "Dumping %d nic records to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	return nil
}

func dumpDracs(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("dracs$%s", curTimeStr))
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := registration.ListDracs(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return errors.Annotate(err, "get all dracs").Err()
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
	logging.Debugf(ctx, "Dumping %d drac records to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	return nil
}

func dumpMachineLSEs(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("machine_lses$%s", curTimeStr))
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := inventory.ListMachineLSEs(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return errors.Annotate(err, "get all machine lses").Err()
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
	logging.Debugf(ctx, "Dumping %d machine lse records to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	return nil
}

func dumpRackLSEs(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("rack_lses$%s", curTimeStr))
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := inventory.ListRackLSEs(ctx, pageSize, startToken, nil, false)
		if err != nil {
			return errors.Annotate(err, "get all rack lses").Err()
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
	logging.Debugf(ctx, "Dumping %d rack lse records to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	return nil
}

func dumpStateRecord(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) error {
	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, ufsDatasetName, fmt.Sprintf("state_records$%s", curTimeStr))
	msgs := make([]proto.Message, 0)
	for startToken := ""; ; {
		res, nextToken, err := state.ListStateRecords(ctx, pageSize, startToken)
		if err != nil {
			return errors.Annotate(err, "get all state records").Err()
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
	logging.Debugf(ctx, "Dumping %d state records to BigQuery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	logging.Debugf(ctx, "Finish dumping successfully")
	return nil
}
