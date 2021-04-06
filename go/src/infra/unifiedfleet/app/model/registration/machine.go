// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

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

// MachineKind is the datastore entity kind Machine.
const MachineKind string = "Machine"

// MachineEntity is a datastore entity that tracks Machine.
type MachineEntity struct {
	_kind string `gae:"$kind,Machine"`
	// ufspb.Machine.Name
	ID               string   `gae:"$id"`
	SerialNumber     string   `gae:"serial_number"`
	KVMID            string   `gae:"kvm_id"`
	KVMPort          string   `gae:"kvm_port"`
	RPMID            string   `gae:"rpm_id"`
	NicIDs           []string `gae:"nic_ids"` // deprecated. Do not use.
	DracID           string   `gae:"drac_id"` // deprecated. Do not use.
	ChromePlatformID string   `gae:"chrome_platform_id"`
	Rack             string   `gae:"rack"`
	Lab              string   `gae:"lab"` // deprecated
	Zone             string   `gae:"zone"`
	Tags             []string `gae:"tags"`
	State            string   `gae:"state"`
	Model            string   `gae:"model"`
	BuildTarget      string   `gae:"build_target"`
	DeviceType       string   `gae:"device_type"`
	Phase            string   `gae:"phase"`
	// ufspb.Machine cannot be directly used as it contains pointer.
	Machine []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled Machine.
func (e *MachineEntity) GetProto() (proto.Message, error) {
	var p ufspb.Machine
	if err := proto.Unmarshal(e.Machine, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newMachineEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.Machine)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty Machine ID").Err()
	}
	machine, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal Machine %s", p).Err()
	}
	return &MachineEntity{
		ID:               p.GetName(),
		SerialNumber:     p.GetSerialNumber(),
		KVMID:            p.GetChromeBrowserMachine().GetKvmInterface().GetKvm(),
		KVMPort:          p.GetChromeBrowserMachine().GetKvmInterface().GetPortName(),
		RPMID:            p.GetChromeBrowserMachine().GetRpmInterface().GetRpm(),
		ChromePlatformID: p.GetChromeBrowserMachine().GetChromePlatform(),
		Rack:             p.GetLocation().GetRack(),
		Zone:             p.GetLocation().GetZone().String(),
		Tags:             p.GetTags(),
		Machine:          machine,
		State:            p.GetResourceState().String(),
		Model:            strings.ToLower(p.GetChromeosMachine().GetModel()),
		BuildTarget:      p.GetChromeosMachine().GetBuildTarget(),
		DeviceType:       p.GetChromeosMachine().GetDeviceType().String(),
		Phase:            p.GetChromeosMachine().GetPhase(),
	}, nil
}

// QueryMachineByPropertyName queries Machine Entity in the datastore
// If keysOnly is true, then only key field is populated in returned machines
func QueryMachineByPropertyName(ctx context.Context, propertyName, id string, keysOnly bool) ([]*ufspb.Machine, error) {
	q := datastore.NewQuery(MachineKind).KeysOnly(keysOnly).FirestoreMode(true)
	var entities []*MachineEntity
	if err := datastore.GetAll(ctx, q.Eq(propertyName, id), &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if len(entities) == 0 {
		logging.Debugf(ctx, "No machines found for the query: %s", id)
		return nil, nil
	}
	machines := make([]*ufspb.Machine, 0, len(entities))
	for _, entity := range entities {
		if keysOnly {
			machine := &ufspb.Machine{
				Name: entity.ID,
			}
			machines = append(machines, machine)
		} else {
			pm, perr := entity.GetProto()
			if perr != nil {
				logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
				continue
			}
			machines = append(machines, pm.(*ufspb.Machine))
		}
	}
	return machines, nil
}

// CreateMachine creates a new machine in datastore.
func CreateMachine(ctx context.Context, machine *ufspb.Machine) (*ufspb.Machine, error) {
	return putMachine(ctx, machine, false)
}

// UpdateMachine updates machine in datastore.
func UpdateMachine(ctx context.Context, machine *ufspb.Machine) (*ufspb.Machine, error) {
	return putMachine(ctx, machine, true)
}

// GetMachine returns machine for the given id from datastore.
func GetMachine(ctx context.Context, id string) (*ufspb.Machine, error) {
	pm, err := ufsds.Get(ctx, &ufspb.Machine{Name: id}, newMachineEntity)
	if err == nil {
		return pm.(*ufspb.Machine), err
	}
	return nil, err
}

func getMachineID(pm proto.Message) string {
	p := pm.(*ufspb.Machine)
	return p.GetName()
}

// BatchGetMachines returns a batch of machines from datastore.
func BatchGetMachines(ctx context.Context, ids []string) ([]*ufspb.Machine, error) {
	protos := make([]proto.Message, len(ids))
	for i, n := range ids {
		protos[i] = &ufspb.Machine{Name: n}
	}
	pms, err := ufsds.BatchGet(ctx, protos, newMachineEntity, getMachineID)
	if err != nil {
		return nil, err
	}
	res := make([]*ufspb.Machine, len(pms))
	for i, pm := range pms {
		res[i] = pm.(*ufspb.Machine)
	}
	return res, nil
}

// ListMachines lists the machines
// Does a query over Machine entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListMachines(ctx context.Context, pageSize int32, pageToken string, filterMap map[string][]interface{}, keysOnly bool) (res []*ufspb.Machine, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, MachineKind, pageSize, pageToken, filterMap, keysOnly)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *MachineEntity, cb datastore.CursorCB) error {
		if keysOnly {
			machine := &ufspb.Machine{
				Name: ent.ID,
			}
			res = append(res, machine)
		} else {
			pm, err := ent.GetProto()
			if err != nil {
				logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
				return nil
			}
			res = append(res, pm.(*ufspb.Machine))
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
		logging.Errorf(ctx, "Failed to List Machines %s", err)
		return nil, "", status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

func queryAllMachine(ctx context.Context) ([]ufsds.FleetEntity, error) {
	var entities []*MachineEntity
	q := datastore.NewQuery(MachineKind)
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	fe := make([]ufsds.FleetEntity, len(entities))
	for i, e := range entities {
		fe[i] = e
	}
	return fe, nil
}

// GetAllMachines returns all machines in datastore.
func GetAllMachines(ctx context.Context) (*ufsds.OpResults, error) {
	return ufsds.GetAll(ctx, queryAllMachine)
}

// ListAllMachines returns all machine in datastore.
func ListAllMachines(ctx context.Context, keysOnly bool) (res []*ufspb.Machine, err error) {
	var entities []*MachineEntity
	q := datastore.NewQuery(MachineKind).KeysOnly(keysOnly).FirestoreMode(true)
	if err = datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	for _, ent := range entities {
		if keysOnly {
			res = append(res, &ufspb.Machine{
				Name: ent.ID,
			})
		} else {
			pm, err := ent.GetProto()
			if err != nil {
				logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
				return nil, err
			}
			machine := pm.(*ufspb.Machine)
			res = append(res, machine)
		}
	}
	return
}

// DeleteMachines deletes a batch of machines
func DeleteMachines(ctx context.Context, resourceNames []string) *ufsds.OpResults {
	protos := make([]proto.Message, len(resourceNames))
	for i, m := range resourceNames {
		protos[i] = &ufspb.Machine{
			Name: m,
		}
	}
	return ufsds.DeleteAll(ctx, protos, newMachineEntity)
}

// DeleteMachine deletes the machine in datastore
func DeleteMachine(ctx context.Context, id string) error {
	return ufsds.Delete(ctx, &ufspb.Machine{Name: id}, newMachineEntity)
}

// ImportMachines creates or updates a batch of machines in datastore
func ImportMachines(ctx context.Context, machines []*ufspb.Machine) (*ufsds.OpResults, error) {
	protos := make([]proto.Message, len(machines))
	utime := ptypes.TimestampNow()
	for i, m := range machines {
		m.UpdateTime = utime
		protos[i] = m
	}
	return ufsds.Insert(ctx, protos, newMachineEntity, true, true)
}

// BatchUpdateMachines updates machines in datastore.
//
// This is a non-atomic operation and doesnt check if the object already exists before
// update. Must be used within a Transaction where objects are checked before update.
// Will lead to partial updates if not used in a transaction.
func BatchUpdateMachines(ctx context.Context, machines []*ufspb.Machine) ([]*ufspb.Machine, error) {
	return putAllMachine(ctx, machines, true)
}

func putMachine(ctx context.Context, machine *ufspb.Machine, update bool) (*ufspb.Machine, error) {
	machine.UpdateTime = ptypes.TimestampNow()
	pm, err := ufsds.Put(ctx, machine, newMachineEntity, update)
	if err == nil {
		return pm.(*ufspb.Machine), err
	}
	return nil, err
}

func putAllMachine(ctx context.Context, machines []*ufspb.Machine, update bool) ([]*ufspb.Machine, error) {
	protos := make([]proto.Message, len(machines))
	updateTime := ptypes.TimestampNow()
	for i, machine := range machines {
		machine.UpdateTime = updateTime
		protos[i] = machine
	}
	_, err := ufsds.PutAll(ctx, protos, newMachineEntity, update)
	if err == nil {
		return machines, err
	}
	return nil, err
}

// GetMachineIndexedFieldName returns the index name
func GetMachineIndexedFieldName(input string) (string, error) {
	var field string
	input = strings.TrimSpace(input)
	switch strings.ToLower(input) {
	case util.SerialNumberFilterName:
		field = "serial_number"
	case util.KVMFilterName:
		field = "kvm_id"
	case util.RPMFilterName:
		field = "rpm_id"
	case util.ZoneFilterName:
		field = "zone"
	case util.RackFilterName:
		field = "rack"
	case util.ChromePlatformFilterName:
		field = "chrome_platform_id"
	case util.TagFilterName:
		field = "tags"
	case util.StateFilterName:
		field = "state"
	case util.KVMPortFilterName:
		field = "kvm_port"
	case util.ModelFilterName:
		field = "model"
	case util.BuildTargetFilterName:
		field = "build_target"
	case util.DeviceTypeFilterName:
		field = "device_type"
	case util.PhaseFilterName:
		field = "phase"
	default:
		return "", status.Errorf(codes.InvalidArgument, "Invalid field name %s - field name for machine are serialnumber/kvm/kvmport/rpm/zone/rack/platform/tag/state/model/buildtarget(target)/devicetype/phase", input)
	}
	return field, nil
}
