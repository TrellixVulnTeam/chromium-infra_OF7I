// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	"context"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
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
	KVMID            string   `gae:"kvm_id"`
	RPMID            string   `gae:"rpm_id"`
	NicIDs           []string `gae:"nic_ids"`
	DracID           string   `gae:"drac_id"`
	ChromePlatformID string   `gae:"chrome_platform_id"`
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
		KVMID:            p.GetChromeBrowserMachine().GetKvmInterface().GetKvm(),
		RPMID:            p.GetChromeBrowserMachine().GetRpmInterface().GetRpm(),
		NicIDs:           p.GetChromeBrowserMachine().GetNics(),
		DracID:           p.GetChromeBrowserMachine().GetDrac(),
		ChromePlatformID: p.GetChromeBrowserMachine().GetChromePlatform(),
		Machine:          machine,
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
		logging.Infof(ctx, "No machines found for the query: %s", id)
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

// ListMachines lists the machines
// Does a query over Machine entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListMachines(ctx context.Context, pageSize int32, pageToken string) (res []*ufspb.Machine, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, MachineKind, pageSize, pageToken, nil, false)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *MachineEntity, cb datastore.CursorCB) error {
		pm, err := ent.GetProto()
		if err != nil {
			logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
			return nil
		}
		res = append(res, pm.(*ufspb.Machine))
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
	case util.KVMFilterName:
		field = "kvm_id"
	case util.RPMFilterName:
		field = "rpm_id"
	case util.NicFilterName:
		field = "nic_ids"
	case util.DracFilterName:
		field = "drac_id"
	case util.LabFilterName:
		field = "lab"
	case util.RackFilterName:
		field = "rack"
	case util.ChromePlatformFilterName:
		field = "chrome_platform_id"
	default:
		return "", status.Errorf(codes.InvalidArgument, "Invalid field name %s - field name for machine are kvm/rpm/drac/nic/lab/rack/platform", input)
	}
	return field, nil
}
