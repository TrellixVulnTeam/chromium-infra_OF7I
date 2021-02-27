// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package converter

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	apibq "infra/appengine/cros/lab_inventory/api/bigquery"
	"infra/appengine/cros/lab_inventory/app/external/ufs"
	ds "infra/cros/lab_inventory/datastore"
)

// DeviceToBQMsgs converts a device to messages that can be committed to bigquery.
func DeviceToBQMsgs(d *ds.DeviceOpResult) (*apibq.LabInventory, *apibq.StateConfigInventory, error) {
	if d == nil || d.Entity == nil {
		return nil, nil, fmt.Errorf("deviceOpResult cannot be empty")
	}
	if d.Err != nil {
		return nil, nil, fmt.Errorf("failed device response: %s", d.Err)
	}
	if d.Entity.ID == "" {
		return nil, nil, fmt.Errorf("Non-existing empty entity ID")
	}

	state := &lab.DutState{}
	if err := d.Entity.GetDutStateProto(state); err != nil {
		return nil, nil, err
	}
	dev := &lab.ChromeOSDevice{}
	if err := d.Entity.GetCrosDeviceProto(dev); err != nil {
		return nil, nil, err
	}

	id := string(d.Entity.ID)
	utime, _ := ptypes.TimestampProto(d.Entity.Updated)
	return &apibq.LabInventory{
			Id:          id,
			Hostname:    d.Entity.Hostname,
			Device:      dev,
			UpdatedTime: utime,
		}, &apibq.StateConfigInventory{
			Id:          id,
			State:       state,
			UpdatedTime: utime,
		}, nil
}

// DeviceToBQMsgsSeq converts a sequence of devices into messages that can be committed to bigquery.
func DeviceToBQMsgsSeq(rs ds.DeviceOpResults) ([]proto.Message, []proto.Message, error) {
	merr := errors.NewMultiError()
	if rs == nil {
		merr = append(merr, fmt.Errorf("deviceOpResult cannot be nil"))
		return nil, nil, merr
	}

	labconfigs := make([]proto.Message, len(rs))
	stateconfigs := make([]proto.Message, len(rs))
	for i, dev := range rs {
		labconfig, stateconfig, err := DeviceToBQMsgs(&dev)
		if err != nil {
			merr = append(merr, err)
			continue
		}
		fmt.Println(labconfig)
		labconfigs[i] = labconfig
		stateconfigs[i] = stateconfig

	}
	// Return outs & err together, don't stop uploading for failures.
	return labconfigs, stateconfigs, merr
}

// DeviceDataToBQDeviceMsgs converts a sequence of devices data into messages that can be committed to bigquery.
func DeviceDataToBQDeviceMsgs(ctx context.Context, devicesData []*ufs.DeviceData) []proto.Message {
	labconfigs := make([]proto.Message, len(devicesData))
	for i, data := range devicesData {
		if data.Device == nil || data.UpdateTime == nil {
			logging.Errorf(ctx, "deviceData Device or UpdateTime is nil")
			continue
		}
		var hostname string
		if data.Device.GetDut() != nil {
			hostname = data.Device.GetDut().GetHostname()
		} else {
			hostname = data.Device.GetLabstation().GetHostname()
		}
		labconfigs[i] = &apibq.LabInventory{
			Id:          data.Device.GetId().GetValue(),
			Hostname:    hostname,
			Device:      data.Device,
			UpdatedTime: data.UpdateTime,
		}
		fmt.Println(labconfigs[i])
	}
	return labconfigs
}

// DutStateDataToBQDutStateMsgs converts a sequence of dutStates data into messages that can be committed to bigquery.
func DutStateDataToBQDutStateMsgs(ctx context.Context, dutStatesData []*ufs.DutStateData) []proto.Message {
	stateconfigs := make([]proto.Message, len(dutStatesData))
	for i, data := range dutStatesData {
		if data.DutState == nil || data.UpdateTime == nil {
			logging.Errorf(ctx, "dutStateData DutState or UpdateTime is nil")
			continue
		}
		stateconfigs[i] = &apibq.StateConfigInventory{
			Id:          data.DutState.GetId().GetValue(),
			State:       data.DutState,
			UpdatedTime: data.UpdateTime,
		}
		fmt.Println(stateconfigs[i])
	}
	return stateconfigs
}
