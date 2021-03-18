// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dumper

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	bqlib "infra/cros/lab_inventory/bq"
	ufspb "infra/unifiedfleet/api/v1/models"
	apibq "infra/unifiedfleet/api/v1/models/bigquery"
	chromeoslab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/util"
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
	logging.Debugf(ctx, "Finish uploading change events successfully")
	logging.Debugf(ctx, "Deleting uploaded entities")
	if err := history.DeleteChangeEventEntities(ctx, changes); err != nil {
		logging.Debugf(ctx, "fail to delete entities: %s", err.Error())
		return err
	}
	logging.Debugf(ctx, "Finish deleting successfully")
	return nil
}

func dumpChangeSnapshotHelper(ctx context.Context, bqClient *bigquery.Client) error {
	snapshots, err := history.GetAllSnapshotMsg(ctx)
	if err != nil {
		return errors.Annotate(err, "get all snapshot msg entities").Err()
	}

	var curTimeStr string
	proConfig, err := configuration.GetProjectConfig(ctx, getProject(ctx))
	if err != nil {
		curTimeStr = bqlib.GetPSTTimeStamp(time.Now())
	} else {
		curTimeStr = proConfig.DailyDumpTimeStr
	}

	msgs := make(map[string][]proto.Message, 0)
	updateUTime := ptypes.TimestampNow()
	for _, s := range snapshots {
		resourceType := util.GetPrefix(s.ResourceName)
		logging.Debugf(ctx, "handling %s", s.ResourceName)
		switch resourceType {
		case util.MachineCollection:
			var data ufspb.Machine
			if err := s.GetProto(&data); err != nil {
				continue
			}
			data.UpdateTime = updateUTime
			msgs["machines"] = append(msgs["machines"], &apibq.MachineRow{
				Machine: &data,
				Delete:  s.Delete,
			})
		case util.NicCollection:
			var data ufspb.Nic
			if err := s.GetProto(&data); err != nil {
				continue
			}
			data.UpdateTime = updateUTime
			msgs["nics"] = append(msgs["nics"], &apibq.NicRow{
				Nic:    &data,
				Delete: s.Delete,
			})
		case util.DracCollection:
			var data ufspb.Drac
			if err := s.GetProto(&data); err != nil {
				continue
			}
			data.UpdateTime = updateUTime
			msgs["dracs"] = append(msgs["dracs"], &apibq.DracRow{
				Drac:   &data,
				Delete: s.Delete,
			})
		case util.RackCollection:
			var data ufspb.Rack
			if err := s.GetProto(&data); err != nil {
				continue
			}
			data.UpdateTime = updateUTime
			msgs["racks"] = append(msgs["racks"], &apibq.RackRow{
				Rack:   &data,
				Delete: s.Delete,
			})
		case util.KVMCollection:
			var data ufspb.KVM
			if err := s.GetProto(&data); err != nil {
				continue
			}
			data.UpdateTime = updateUTime
			msgs["kvms"] = append(msgs["kvms"], &apibq.KVMRow{
				Kvm:    &data,
				Delete: s.Delete,
			})
		case util.SwitchCollection:
			var data ufspb.Switch
			if err := s.GetProto(&data); err != nil {
				continue
			}
			data.UpdateTime = updateUTime
			msgs["switches"] = append(msgs["switches"], &apibq.SwitchRow{
				Switch: &data,
				Delete: s.Delete,
			})
		case util.HostCollection:
			var data ufspb.MachineLSE
			if err := s.GetProto(&data); err != nil {
				continue
			}
			data.UpdateTime = updateUTime
			msgs["machine_lses"] = append(msgs["machine_lses"], &apibq.MachineLSERow{
				MachineLse: &data,
				Delete:     s.Delete,
			})
		case util.VMCollection:
			var data ufspb.VM
			if err := s.GetProto(&data); err != nil {
				continue
			}
			data.UpdateTime = updateUTime
			msgs["vms"] = append(msgs["vms"], &apibq.VMRow{
				Vm:     &data,
				Delete: s.Delete,
			})
		case util.DHCPCollection:
			var data ufspb.DHCPConfig
			if err := s.GetProto(&data); err != nil {
				continue
			}
			data.UpdateTime = updateUTime
			msgs["dhcps"] = append(msgs["dhcps"], &apibq.DHCPConfigRow{
				DhcpConfig: &data,
				Delete:     s.Delete,
			})
		case util.StateCollection:
			var data ufspb.StateRecord
			if err := s.GetProto(&data); err != nil {
				continue
			}
			data.UpdateTime = updateUTime
			msgs["state_records"] = append(msgs["state_records"], &apibq.StateRecordRow{
				StateRecord: &data,
				Delete:      s.Delete,
			})
		case util.DutStateCollection:
			var data chromeoslab.DutState
			if err := s.GetProto(&data); err != nil {
				continue
			}
			data.UpdateTime = updateUTime
			msgs["dutstates"] = append(msgs["dutstates"], &apibq.DUTStateRecordRow{
				State: &data,
			})
		}
	}
	logging.Debugf(ctx, "Uploading all %d snapshots...", len(snapshots))
	for tableName, ms := range msgs {
		if err := dumpHelper(ctx, bqClient, ms, tableName, curTimeStr); err != nil {
			return err
		}
	}
	logging.Debugf(ctx, "Finish uploading the snapshots successfully")
	logging.Debugf(ctx, "Deleting the uploaded snapshots")
	if err := history.DeleteSnapshotMsgEntities(ctx, snapshots); err != nil {
		logging.Debugf(ctx, "fail to delete snapshot msg entities: %s", err.Error())
		return err
	}
	logging.Debugf(ctx, "Finish deleting the snapshots successfully")
	return nil
}

func dumpConfigurations(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) (err error) {
	for k, f := range configurationDumpToolkit {
		logging.Infof(ctx, "dumping %s", k)
		msgs, err := f(ctx)
		if err != nil {
			return err
		}
		if err := dumpHelper(ctx, bqClient, msgs, k, curTimeStr); err != nil {
			return err
		}
	}
	return nil
}

func dumpRegistration(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) (err error) {
	for k, f := range registrationDumpToolkit {
		logging.Infof(ctx, "dumping %s", k)
		msgs, err := f(ctx)
		if err != nil {
			return err
		}
		if err := dumpHelper(ctx, bqClient, msgs, k, curTimeStr); err != nil {
			return err
		}
	}
	return nil
}

func dumpInventory(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) (err error) {
	for k, f := range inventoryDumpToolkit {
		logging.Infof(ctx, "dumping %s", k)
		msgs, err := f(ctx)
		if err != nil {
			return err
		}
		if err := dumpHelper(ctx, bqClient, msgs, k, curTimeStr); err != nil {
			return err
		}
	}
	return nil
}

func dumpState(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) (err error) {
	for k, f := range stateDumpToolkit {
		logging.Infof(ctx, "dumping %s", k)
		msgs, err := f(ctx)
		if err != nil {
			return err
		}
		if err := dumpHelper(ctx, bqClient, msgs, k, curTimeStr); err != nil {
			return err
		}
	}
	return nil
}
