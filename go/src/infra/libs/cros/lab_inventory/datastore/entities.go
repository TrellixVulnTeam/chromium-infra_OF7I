// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package datastore

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/gae/service/datastore"

	"infra/libs/cros/lab_inventory/changehistory"
	"infra/libs/cros/lab_inventory/utils"
)

// DeviceEntityID represents the ID of a device. We prefer use asset id as the id.
type DeviceEntityID string

// DeviceKind is the datastore entity kind for Device entities.
const DeviceKind string = "Device"

// DeviceEntity is a datastore entity that tracks a device.
type DeviceEntity struct {
	_kind     string         `gae:"$kind,Device"`
	ID        DeviceEntityID `gae:"$id"`
	Hostname  string
	LabConfig []byte `gae:",noindex"`
	DutState  []byte `gae:",noindex"`
	Updated   time.Time
	Parent    *datastore.Key `gae:"$parent"`
}

// GetCrosDeviceProto gets the unmarshaled proto message data.
func (e *DeviceEntity) GetCrosDeviceProto(p *lab.ChromeOSDevice) error {
	if err := proto.Unmarshal(e.LabConfig, p); err != nil {
		return err
	}
	return nil
}

// GetDutStateProto gets the unmarshaled proto message data.
func (e *DeviceEntity) GetDutStateProto(p *lab.DutState) error {
	if err := proto.Unmarshal(e.DutState, p); err != nil {
		return err
	}
	return nil
}

func (e *DeviceEntity) updateLabConfig(p *lab.ChromeOSDevice) (changehistory.Changes, error) {
	var oldMsg lab.ChromeOSDevice
	if err := proto.Unmarshal(e.LabConfig, &oldMsg); err != nil {
		return nil, err
	}
	if proto.Equal(p, &oldMsg) {
		// Do nothing if the proto message is identical.
		return nil, nil
	}
	data, err := proto.Marshal(p)
	if err != nil {
		return nil, err
	}
	changes := changehistory.LogChromeOSDeviceChanges(p, &oldMsg)

	e.LabConfig = data
	e.Hostname = utils.GetHostname(p)

	return changes, nil
}

func (e *DeviceEntity) updateDutState(p *lab.DutState) (changehistory.Changes, error) {
	var oldMsg lab.DutState
	if err := proto.Unmarshal(e.DutState, &oldMsg); err != nil {
		return nil, err
	}
	if proto.Equal(p, &oldMsg) {
		// Do nothing if the proto message is identical.
		return nil, nil
	}
	data, err := proto.Marshal(p)
	if err != nil {
		return nil, err
	}
	changes := changehistory.LogDutStateChanges(e.Hostname, p, &oldMsg)

	e.DutState = data
	return changes, nil
}

// UpdatePayload sets the proto data to the entity.
func (e *DeviceEntity) UpdatePayload(p proto.Message, t time.Time) (changes changehistory.Changes, err error) {
	if v, ok := p.(*lab.ChromeOSDevice); ok {
		changes, err = e.updateLabConfig(v)
	} else if v, ok := p.(*lab.DutState); ok {
		changes, err = e.updateDutState(v)
	}
	e.Updated = t
	return
}

func (e *DeviceEntity) String() string {
	return fmt.Sprintf("<%s:%s>", e.Hostname, e.ID)
}
