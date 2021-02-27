// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/models"
	ufsds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/util"
)

// MachineLSEKind is the datastore entity kind MachineLSE.
const MachineLSEKind string = "MachineLSE"

// MachineLSEEntity is a datastore entity that tracks MachineLSE.
type MachineLSEEntity struct {
	_kind                 string   `gae:"$kind,MachineLSE"`
	ID                    string   `gae:"$id"`
	MachineIDs            []string `gae:"machine_ids"`
	MachineLSEProtoTypeID string   `gae:"machinelse_prototype_id"`
	SwitchID              string   `gae:"switch_id"`
	RPMID                 string   `gae:"rpm_id"`
	RPMPort               string   `gae:"rpm_port"`
	VlanID                string   `gae:"vlan_id"`
	ServoID               string   `gae:"servo_id"`
	ServoType             string   `gae:"servo_type"`
	Rack                  string   `gae:"rack"`
	Lab                   string   `gae:"lab"` // deprecated
	Zone                  string   `gae:"zone"`
	Manufacturer          string   `gae:"manufacturer"`
	Tags                  []string `gae:"tags"`
	State                 string   `gae:"state"`
	OS                    []string `gae:"os"`
	VirtualDatacenter     string   `gae:"virtualdatacenter"`
	Nic                   string   `gae:"nic"`
	Pools                 []string `gae:"pools"`
	// ufspb.MachineLSE cannot be directly used as it contains pointer.
	MachineLSE []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled MachineLSE.
func (e *MachineLSEEntity) GetProto() (proto.Message, error) {
	var p ufspb.MachineLSE
	if err := proto.Unmarshal(e.MachineLSE, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newMachineLSEEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.MachineLSE)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty MachineLSE ID").Err()
	}
	machineLSE, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal MachineLSE %s", p).Err()
	}
	servo := p.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
	servoID := ufsds.GetServoID(servo.GetServoHostname(), servo.GetServoPort())
	var rpmID string
	var rpmPort string
	var pools []string
	if p.GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
		rpmID = p.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetRpm().GetPowerunitName()
		rpmPort = p.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetRpm().GetPowerunitOutlet()
		pools = p.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPools()
	} else if p.GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		rpmID = p.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetRpm().GetPowerunitOutlet()
		rpmPort = p.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetRpm().GetPowerunitOutlet()
		pools = p.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetPools()
	}

	return &MachineLSEEntity{
		ID:                    p.GetName(),
		MachineIDs:            p.GetMachines(),
		MachineLSEProtoTypeID: p.GetMachineLsePrototype(),
		SwitchID:              p.GetChromeosMachineLse().GetDeviceLse().GetNetworkDeviceInterface().GetSwitch(),
		RPMID:                 rpmID,
		RPMPort:               rpmPort,
		VlanID:                p.GetVlan(),
		ServoID:               servoID,
		ServoType:             servo.GetServoType(),
		Rack:                  p.GetRack(),
		Zone:                  p.GetZone(),
		Manufacturer:          strings.ToLower(p.GetManufacturer()),
		State:                 p.GetResourceState().String(),
		OS:                    ufsds.GetOSIndex(p.GetChromeBrowserMachineLse().GetOsVersion().GetValue()),
		VirtualDatacenter:     p.GetChromeBrowserMachineLse().GetVirtualDatacenter(),
		Nic:                   p.GetNic(),
		Tags:                  p.GetTags(),
		Pools:                 pools,
		MachineLSE:            machineLSE,
	}, nil
}

// QueryMachineLSEByPropertyName queries MachineLSE Entity in the datastore
// If keysOnly is true, then only key field is populated in returned machinelses
func QueryMachineLSEByPropertyName(ctx context.Context, propertyName, id string, keysOnly bool) ([]*ufspb.MachineLSE, error) {
	return QueryMachineLSEByPropertyNames(ctx, map[string]string{propertyName: id}, keysOnly)
}

// QueryMachineLSEByPropertyNames queries MachineLSE Entity in the datastore
// If keysOnly is true, then only key field is populated in returned machinelses
func QueryMachineLSEByPropertyNames(ctx context.Context, propertyMap map[string]string, keysOnly bool) ([]*ufspb.MachineLSE, error) {
	q := datastore.NewQuery(MachineLSEKind).KeysOnly(keysOnly).FirestoreMode(true)
	var entities []*MachineLSEEntity
	for propertyName, id := range propertyMap {
		q = q.Eq(propertyName, id)
	}
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if len(entities) == 0 {
		logging.Infof(ctx, "No machineLSEs found for the query: %s", q.String())
		return nil, nil
	}
	machineLSEs := make([]*ufspb.MachineLSE, 0, len(entities))
	for _, entity := range entities {
		if keysOnly {
			machineLSE := &ufspb.MachineLSE{
				Name: entity.ID,
			}
			machineLSEs = append(machineLSEs, machineLSE)
		} else {
			pm, perr := entity.GetProto()
			if perr != nil {
				logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
				continue
			}
			machineLSEs = append(machineLSEs, pm.(*ufspb.MachineLSE))
		}
	}
	return machineLSEs, nil
}

// CreateMachineLSE creates a new machineLSE in datastore.
func CreateMachineLSE(ctx context.Context, machineLSE *ufspb.MachineLSE) (*ufspb.MachineLSE, error) {
	return putMachineLSE(ctx, machineLSE, false)
}

// UpdateMachineLSE updates machineLSE in datastore.
func UpdateMachineLSE(ctx context.Context, machineLSE *ufspb.MachineLSE) (*ufspb.MachineLSE, error) {
	return putMachineLSE(ctx, machineLSE, true)
}

// GetMachineLSE returns machine for the given id from datastore.
func GetMachineLSE(ctx context.Context, id string) (*ufspb.MachineLSE, error) {
	pm, err := ufsds.Get(ctx, &ufspb.MachineLSE{Name: id}, newMachineLSEEntity)
	if err == nil {
		return pm.(*ufspb.MachineLSE), err
	}
	return nil, err
}

func getLSEID(pm proto.Message) string {
	p := pm.(*ufspb.MachineLSE)
	return p.GetName()
}

// BatchGetMachineLSEs returns a batch of machine lses from datastore.
func BatchGetMachineLSEs(ctx context.Context, ids []string) ([]*ufspb.MachineLSE, error) {
	protos := make([]proto.Message, len(ids))
	for i, n := range ids {
		protos[i] = &ufspb.MachineLSE{Name: n}
	}
	pms, err := ufsds.BatchGet(ctx, protos, newMachineLSEEntity, getLSEID)
	if err != nil {
		return nil, err
	}
	res := make([]*ufspb.MachineLSE, len(pms))
	for i, pm := range pms {
		res[i] = pm.(*ufspb.MachineLSE)
	}
	return res, nil
}

// ListMachineLSEs lists the machine lses
// Does a query over MachineLSE entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListMachineLSEs(ctx context.Context, pageSize int32, pageToken string, filterMap map[string][]interface{}, keysOnly bool) (res []*ufspb.MachineLSE, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, MachineLSEKind, pageSize, pageToken, filterMap, keysOnly)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *MachineLSEEntity, cb datastore.CursorCB) error {
		if keysOnly {
			res = append(res, &ufspb.MachineLSE{
				Name: ent.ID,
			})
		} else {
			pm, err := ent.GetProto()
			if err != nil {
				logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
				return nil
			}
			machineLSE := pm.(*ufspb.MachineLSE)
			res = append(res, machineLSE)
		}
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to List MachineLSEs %s", err)
		return nil, "", status.Errorf(codes.Internal, err.Error())
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// ListFreeMachineLSEs lists the machine lses with vm capacity
func ListFreeMachineLSEs(ctx context.Context, requiredSize int32, filterMap map[string][]interface{}, capacityMap map[string]int) (res []*ufspb.MachineLSE, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, MachineLSEKind, -1, "", filterMap, false)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *MachineLSEEntity, cb datastore.CursorCB) error {
		pm, err := ent.GetProto()
		if err != nil {
			logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
			return nil
		}
		machineLSE := pm.(*ufspb.MachineLSE)
		if machineLSE.GetChromeBrowserMachineLse().GetVmCapacity() > int32(capacityMap[machineLSE.GetName()]) {
			res = append(res, machineLSE)
		}
		if len(res) >= int(requiredSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to List MachineLSEs %s", err)
		return nil, "", status.Errorf(codes.Internal, err.Error())
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// ListAllMachineLSEs returns all machine lses in datastore.
func ListAllMachineLSEs(ctx context.Context, keysOnly bool) (res []*ufspb.MachineLSE, err error) {
	var entities []*MachineLSEEntity
	q := datastore.NewQuery(MachineLSEKind).KeysOnly(keysOnly).FirestoreMode(true)
	if err = datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	for _, ent := range entities {
		if keysOnly {
			res = append(res, &ufspb.MachineLSE{
				Name: ent.ID,
			})
		} else {
			pm, err := ent.GetProto()
			if err != nil {
				logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
				return nil, err
			}
			machineLSE := pm.(*ufspb.MachineLSE)
			res = append(res, machineLSE)
		}
	}
	return
}

// DeleteMachineLSE deletes the machineLSE in datastore
func DeleteMachineLSE(ctx context.Context, id string) error {
	return ufsds.Delete(ctx, &ufspb.MachineLSE{Name: id}, newMachineLSEEntity)
}

// BatchUpdateMachineLSEs updates machineLSEs in datastore.
// This is a non-atomic operation and doesnt check if the object already exists before
// update. Must be used within a Transaction where objects are checked before update.
// Will lead to partial updates if not used in a transaction.
func BatchUpdateMachineLSEs(ctx context.Context, machineLSEs []*ufspb.MachineLSE) ([]*ufspb.MachineLSE, error) {
	return putAllMachineLSE(ctx, machineLSEs, true)
}

func putMachineLSE(ctx context.Context, machineLSE *ufspb.MachineLSE, update bool) (*ufspb.MachineLSE, error) {
	machineLSE.UpdateTime = ptypes.TimestampNow()
	pm, err := ufsds.Put(ctx, machineLSE, newMachineLSEEntity, update)
	if err != nil {
		return nil, errors.Annotate(err, "put machine LSE").Err()
	}
	return pm.(*ufspb.MachineLSE), err
}

func putAllMachineLSE(ctx context.Context, machineLSEs []*ufspb.MachineLSE, update bool) ([]*ufspb.MachineLSE, error) {
	protos := make([]proto.Message, len(machineLSEs))
	updateTime := ptypes.TimestampNow()
	for i, machineLSE := range machineLSEs {
		machineLSE.UpdateTime = updateTime
		protos[i] = machineLSE
	}
	_, err := ufsds.PutAll(ctx, protos, newMachineLSEEntity, update)
	if err == nil {
		return machineLSEs, err
	}
	return nil, err
}

// ImportMachineLSEs creates or updates a batch of machine lses in datastore
func ImportMachineLSEs(ctx context.Context, lses []*ufspb.MachineLSE) (*ufsds.OpResults, error) {
	protos := make([]proto.Message, len(lses))
	utime := ptypes.TimestampNow()
	for i, m := range lses {
		if m.UpdateTime == nil {
			m.UpdateTime = utime
		}
		protos[i] = m
	}
	return ufsds.Insert(ctx, protos, newMachineLSEEntity, true, true)
}

func queryAllMachineLSE(ctx context.Context) ([]ufsds.FleetEntity, error) {
	var entities []*MachineLSEEntity
	q := datastore.NewQuery(MachineLSEKind)
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	fe := make([]ufsds.FleetEntity, len(entities))
	for i, e := range entities {
		fe[i] = e
	}
	return fe, nil
}

// GetAllMachineLSEs returns all machine lses in datastore.
func GetAllMachineLSEs(ctx context.Context) (*ufsds.OpResults, error) {
	return ufsds.GetAll(ctx, queryAllMachineLSE)
}

// DeleteMachineLSEs deletes a batch of machine LSEs
func DeleteMachineLSEs(ctx context.Context, resourceNames []string) *ufsds.OpResults {
	protos := make([]proto.Message, len(resourceNames))
	for i, m := range resourceNames {
		protos[i] = &ufspb.MachineLSE{
			Name: m,
		}
	}
	return ufsds.DeleteAll(ctx, protos, newMachineLSEEntity)
}

// GetMachineLSEIndexedFieldName returns the index name
func GetMachineLSEIndexedFieldName(input string) (string, error) {
	var field string
	input = strings.TrimSpace(input)
	switch strings.ToLower(input) {
	case util.SwitchFilterName:
		field = "switch_id"
	case util.RPMFilterName:
		field = "rpm_id"
	case util.RPMPortFilterName:
		field = "rpm_port"
	case util.VlanFilterName:
		field = "vlan_id"
	case util.ServoFilterName:
		field = "servo_id"
	case util.ServoTypeFilterName:
		field = "servo_type"
	case util.ZoneFilterName:
		field = "zone"
	case util.RackFilterName:
		field = "rack"
	case util.MachineFilterName:
		field = "machine_ids"
	case util.MachinePrototypeFilterName:
		field = "machinelse_prototype_id"
	case util.ManufacturerFilterName:
		field = "manufacturer"
	case util.FreeVMFilterName:
		field = "free"
	case util.TagFilterName:
		field = "tags"
	case util.StateFilterName:
		field = "state"
	case util.OSFilterName:
		field = "os"
	case util.VirtualDatacenterFilterName:
		field = "virtualdatacenter"
	case util.NicFilterName:
		field = "nic"
	case util.PoolsFilterName:
		field = "pools"
	default:
		return "", status.Errorf(codes.InvalidArgument, "Invalid field name %s - field name for host are nic/machine/machineprototype/rpm/rpmport/vlan/servo/servotype/zone/rack/switch/man/free/tag/state/os/vdc(virtualdatacenter)/pools", input)
	}
	return field, nil
}
