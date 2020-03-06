// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package converter

import (
	"fmt"

	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/luci/common/errors"

	apibq "infra/appengine/cros/lab_inventory/api/bigquery"
	ds "infra/libs/cros/lab_inventory/datastore"
)

// ToBQLabInventory converts a DeviceOpResult to a LabInventory item with
// the same content.
func ToBQLabInventory(d *ds.DeviceOpResult) (*apibq.LabInventory, error) {
	dev := &lab.ChromeOSDevice{}
	id := ""
	hostname := ""
	if d == nil {
		return nil, fmt.Errorf("deviceOpResult cannot be nil")
	}
	if d.Err != nil {
		return nil, fmt.Errorf("failed device response: %s", d.Err)
	}
	if d.Entity != nil {
		id = string(d.Entity.ID)
		hostname = d.Entity.Hostname
		if d.Entity.LabConfig == nil {
			return nil, fmt.Errorf("labConfig cannot be nil")
		}
		if err := d.Entity.GetCrosDeviceProto(dev); err != nil {
			return nil, fmt.Errorf("get device proto: %s", err)
		}
	}
	utime, _ := ptypes.TimestampProto(d.Entity.Updated)
	return &apibq.LabInventory{
		Id:          id,
		Hostname:    hostname,
		Device:      dev,
		UpdatedTime: utime,
	}, nil
}

// ToBQLabInventorySeq converts a DeviceOpResults into a sequence of
// items that can be committed to bigquery.
func ToBQLabInventorySeq(rs ds.DeviceOpResults) ([]*apibq.LabInventory, error) {
	if rs == nil {
		return nil, fmt.Errorf("deviceOpResult cannot be nil")
	}
	out := make([]*apibq.LabInventory, len(rs))
	merr := errors.NewMultiError()
	for i, dev := range rs {
		item, err := ToBQLabInventory(&dev)
		if err != nil {
			merr = append(merr, err)
			continue
		}
		out[i] = item
	}
	if len(merr) == 0 {
		return out, nil
	}
	return nil, merr
}
