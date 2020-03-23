// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package converter

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/luci/common/errors"

	apibq "infra/appengine/cros/lab_inventory/api/bigquery"
	ds "infra/libs/cros/lab_inventory/datastore"
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
