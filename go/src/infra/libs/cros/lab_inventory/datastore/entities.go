// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package datastore

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"

	inv "infra/appengine/cros/lab_inventory/api/v1"
	"infra/libs/cros/lab_inventory/changehistory"
	"infra/libs/cros/lab_inventory/utils"
	fleet "infra/libs/fleet/protos"
	ufs "infra/libs/fleet/protos/go"
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
	changes := changehistory.LogChromeOSDeviceChanges(&oldMsg, p)

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
	changes := changehistory.LogDutStateChanges(e.Hostname, &oldMsg, p)

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
	_kind    string `gae:"$kind,Asset"`
	ID       string `gae:"$id"`
	Lab      string
	Location []byte         `gae:",noindex"`
	Parent   *datastore.Key `gae:"$parent"`
}

// AssetStateEntityName is the datastore entity kind for Asset state entities.
const AssetStateEntityName string = "AssetState"

// AssetStateEntity is the datastore that tracks the asset state.
type AssetStateEntity struct {
	_kind   string           `gae:"$kind,AssetState"`
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
	var location ufs.Location
	err := proto.Unmarshal(e.Location, &location)
	return &fleet.ChopsAsset{
		Id:       e.ID,
		Location: &location,
	}, err
}

/* Asset Entity and helper functions end */

/* Asset Info and helper funtions */

// AssetInfoEntity is a datastore entity that tracks the asset info from HaRT.
type AssetInfoEntity struct {
	_kind    string        `gae:"$kind,AssetInfo"`
	AssetTag string        `gae:"$id"`
	Info     ufs.AssetInfo `gae:",noindex"`
}

// AssetInfoEntityKind is the datastore entity kind for AssetInfo entities.
const AssetInfoEntityKind = "AssetInfo"

// NewAssetInfo creates an AssetInfoEntity object from AssetInfo object
func NewAssetInfo(a *ufs.AssetInfo) (*AssetInfoEntity, error) {
	if a.GetAssetTag() == "" {
		return nil, errors.Reason("Missing asset tag").Err()
	}
	return &AssetInfoEntity{
		AssetTag: a.GetAssetTag(),
		Info:     *a,
	}, nil
}

/* Asset Info and helper functions end */

/* Device Manual Repair Record Entity and helper functions */

// DeviceManualRepairRecordEntity is a datastore entity that tracks a manual
// repair record of a device.
//
// Possible RepairState based on proto enum:
//  STATE_INVALID = 0;
// 	STATE_NOT_STARTED = 1;
// 	STATE_IN_PROGRESS = 2;
// 	STATE_COMPLETED = 3;
type DeviceManualRepairRecordEntity struct {
	_kind       string `gae:"$kind,DeviceManualRepairRecord"`
	ID          string `gae:"$id"`
	Hostname    string `gae:"hostname"`
	AssetTag    string `gae:"asset_tag"`
	RepairState string `gae:"repair_state"`
	Content     []byte `gae:",noindex"`
}

// DeviceManualRepairRecordEntityKind is the datastore entity kind for
// DeviceManualRepairRecord entities.
const DeviceManualRepairRecordEntityKind = "DeviceManualRepairRecord"

// NewDeviceManualRepairRecordEntity creates a new
// DeviceManualRepairRecordEntity from a DeviceManualRepairRecord object.
func NewDeviceManualRepairRecordEntity(r *inv.DeviceManualRepairRecord) (*DeviceManualRepairRecordEntity, error) {
	hostname := r.GetHostname()
	assetTag := r.GetAssetTag()
	createdTime := ptypes.TimestampString(r.GetCreatedTime())

	if hostname == "" {
		return nil, errors.Reason("Hostname cannot be empty").Err()
	} else if assetTag == "" {
		return nil, errors.Reason("Asset Tag cannot be empty").Err()
	} else if createdTime == "" {
		return nil, errors.Reason("Created Time cannot be empty").Err()
	}
	content, err := proto.Marshal(r)
	if err != nil {
		return nil, err
	}

	id, err := generateRepairRecordID(hostname, assetTag, createdTime)
	if err != nil {
		return nil, err
	}

	return &DeviceManualRepairRecordEntity{
		ID:          id,
		Hostname:    hostname,
		AssetTag:    assetTag,
		RepairState: r.GetRepairState().String(),
		Content:     content,
	}, nil
}

// UpdateDeviceManualRepairRecordEntity sets the proto data to the entity.
func (e *DeviceManualRepairRecordEntity) UpdateDeviceManualRepairRecordEntity(r *inv.DeviceManualRepairRecord) error {
	var oldMsg inv.DeviceManualRepairRecord
	if err := proto.Unmarshal(e.Content, &oldMsg); err != nil {
		return err
	}

	if r.GetHostname() == "" {
		return errors.Reason("Hostname cannot be empty").Err()
	} else if r.GetAssetTag() == "" {
		return errors.Reason("Asset Tag cannot be empty").Err()
	} else if ptypes.TimestampString(r.GetCreatedTime()) == "" {
		return errors.Reason("Created Time cannot be empty").Err()
	}

	// Update record if the message is different from the old one.
	if !proto.Equal(r, &oldMsg) {
		data, err := proto.Marshal(r)
		if err != nil {
			return err
		}
		e.RepairState = r.GetRepairState().String()
		e.Content = data
	}

	return nil
}

// Returns the predefined ID format of $hostname-$assetTag-$createdTime
func generateRepairRecordID(hostname string, assetTag string, createdTime string) (string, error) {
	var err error
	if hostname == "" {
		err = errors.Reason("Hostname cannot be empty").Err()
	} else if assetTag == "" {
		err = errors.Reason("Asset Tag cannot be empty").Err()
	} else if createdTime == "" {
		err = errors.Reason("Created Time cannot be empty").Err()
	}

	return hostname + "-" + assetTag + "-" + createdTime, err
}

/* Device Manual Repair Record Entity and helper functions end */
