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
	"go.chromium.org/luci/common/errors"

	"infra/libs/cros/lab_inventory/changehistory"
	"infra/libs/cros/lab_inventory/utils"
	"infra/libs/fleet/protos"
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

/* Asset Entity and helper functions*/

// AssetEntityName is the datastore entity kind for Asset entities.
const AssetEntityName string = "Asset"

// AssetEntity is a datastore entity that tracks the asset.
type AssetEntity struct {
	_kind    string `gae:"$kind,asset"`
	ID       string `gae:"$id"`
	Lab      string
	Location []byte         `gae:",noindex"`
	Parent   *datastore.Key `gae:"$parent"`
}

// AssetStateEntityName is the datastore entity kind for Asset state entities.
const AssetStateEntityName string = "AssetState"

// AssetStateEntity is the datastore that tracks the asset state.
type AssetStateEntity struct {
	_kind   string           `gae:"$kind,assetstate"`
	ID      string           `gae:"$id"`
	State   fleet.AssetState `gae:",noindex"`
	Updated time.Time
	Parent  *datastore.Key `gae:"$parent"`
}

func (e *AssetEntity) String() string {
	return fmt.Sprintf("<%s>:%s", e.ID, e.Lab)
}

// NewAssetEntity creates an AssetEntity object from ChopsAsset object
func NewAssetEntity(a *fleet.ChopsAsset, parent *datastore.Key) (*AssetEntity, error) {
	if a.GetId() == "" {
		return nil, errors.Reason("Missing asset tag").Err()
	}
	location, err := proto.Marshal(a.GetLocation())
	if err != nil {
		return nil, err
	}
	return &AssetEntity{
		ID:       a.GetId(),
		Lab:      a.GetLocation().GetLab(),
		Location: location,
		Parent:   parent,
	}, nil
}

// NewAssetStateEntity creates an AssetStateEntity object based on input.
func NewAssetStateEntity(a *fleet.ChopsAsset, state fleet.State, updated time.Time, parent *datastore.Key) (*AssetStateEntity, error) {
	if a.GetId() == "" {
		return nil, errors.Reason("Missing asset tag").Err()
	}
	return &AssetStateEntity{
		ID: a.GetId(),
		State: fleet.AssetState{
			Id:    a.GetId(),
			State: state,
		},
		Updated: updated,
		Parent:  parent,
	}, nil
}

// ToChopsAsset returns a ChopsAsset object
func (e *AssetEntity) ToChopsAsset() (*fleet.ChopsAsset, error) {
	var location fleet.Location
	err := proto.Unmarshal(e.Location, &location)
	return &fleet.ChopsAsset{
		Id:       e.ID,
		Location: &location,
	}, err
}

/* Asset Entity and helper functions end */
