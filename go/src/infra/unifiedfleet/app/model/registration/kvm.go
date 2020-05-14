// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fleet "infra/unifiedfleet/api/v1/proto"
	fleetds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
)

// KVMKind is the datastore entity kind KVM.
const KVMKind string = "KVM"

// KVMEntity is a datastore entity that tracks KVM.
type KVMEntity struct {
	_kind            string `gae:"$kind,KVM"`
	ID               string `gae:"$id"`
	ChromePlatformID string `gae:"chrome_platform_id"`
	// fleet.KVM cannot be directly used as it contains pointer.
	KVM []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled KVM.
func (e *KVMEntity) GetProto() (proto.Message, error) {
	var p fleet.KVM
	if err := proto.Unmarshal(e.KVM, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newKVMEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.KVM)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty KVM ID").Err()
	}
	kvm, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal KVM %s", p).Err()
	}
	return &KVMEntity{
		ID:               p.GetName(),
		ChromePlatformID: p.GetChromePlatform(),
		KVM:              kvm,
	}, nil
}

// QueryKVMByPropertyName query's KVM Entity in the datastore
//
// If keysOnly is true, then only key field is populated in returned kvms
func QueryKVMByPropertyName(ctx context.Context, propertyName, id string, keysOnly bool) ([]*fleet.KVM, error) {
	q := datastore.NewQuery(KVMKind).KeysOnly(keysOnly)
	var entities []*KVMEntity
	if err := datastore.GetAll(ctx, q.Eq(propertyName, id), &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if len(entities) == 0 {
		logging.Infof(ctx, "No kvms found for the query: %s", id)
		return nil, nil
	}
	kvms := make([]*fleet.KVM, 0, len(entities))
	for _, entity := range entities {
		if keysOnly {
			kvm := &fleet.KVM{
				Name: entity.ID,
			}
			kvms = append(kvms, kvm)
		} else {
			pm, perr := entity.GetProto()
			if perr != nil {
				logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
				continue
			}
			kvms = append(kvms, pm.(*fleet.KVM))
		}
	}
	return kvms, nil
}

// CreateKVM creates a new KVM in datastore.
func CreateKVM(ctx context.Context, KVM *fleet.KVM) (*fleet.KVM, error) {
	return putKVM(ctx, KVM, false)
}

// UpdateKVM updates KVM in datastore.
func UpdateKVM(ctx context.Context, KVM *fleet.KVM) (*fleet.KVM, error) {
	return putKVM(ctx, KVM, true)
}

// GetKVM returns KVM for the given id from datastore.
func GetKVM(ctx context.Context, id string) (*fleet.KVM, error) {
	pm, err := fleetds.Get(ctx, &fleet.KVM{Name: id}, newKVMEntity)
	if err == nil {
		return pm.(*fleet.KVM), err
	}
	return nil, err
}

// ListKVMs lists the KVMs
//
// Does a query over KVM entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListKVMs(ctx context.Context, pageSize int32, pageToken string) (res []*fleet.KVM, nextPageToken string, err error) {
	q, err := fleetds.ListQuery(ctx, KVMKind, pageSize, pageToken)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *KVMEntity, cb datastore.CursorCB) error {
		pm, err := ent.GetProto()
		if err != nil {
			logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
			return nil
		}
		res = append(res, pm.(*fleet.KVM))
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to list KVMs %s", err)
		return nil, "", status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteKVM deletes the KVM in datastore
//
// For referential data intergrity,
// Delete if there are no references to the KVM by Machine in the datastore.
// If there are any references, delete will be rejected and an error message will be thrown.
func DeleteKVM(ctx context.Context, id string) error {
	machines, err := QueryMachineByPropertyName(ctx, "kvm_id", id, true)
	if err != nil {
		return err
	}
	racks, err := QueryRackByPropertyName(ctx, "kvm_ids", id, true)
	if err != nil {
		return err
	}
	racklses, err := inventory.QueryRackLSEByPropertyName(ctx, "kvm_ids", id, true)
	if err != nil {
		return err
	}
	if len(machines) > 0 || len(racks) > 0 || len(racklses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("KVM %s cannot be deleted because there are other resources which are referring this KVM.", id))
		if len(machines) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nMachines referring the KVM:\n"))
			for _, machine := range machines {
				errorMsg.WriteString(machine.Name + ", ")
			}
		}
		if len(racks) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nRacks referring the KVM:\n"))
			for _, rack := range racks {
				errorMsg.WriteString(rack.Name + ", ")
			}
		}
		if len(racklses) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nRackLSEs referring the KVM:\n"))
			for _, racklse := range racklses {
				errorMsg.WriteString(racklse.Name + ", ")
			}
		}
		logging.Infof(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return fleetds.Delete(ctx, &fleet.KVM{Name: id}, newKVMEntity)
}

func putKVM(ctx context.Context, KVM *fleet.KVM, update bool) (*fleet.KVM, error) {
	KVM.UpdateTime = ptypes.TimestampNow()
	pm, err := fleetds.Put(ctx, KVM, newKVMEntity, update)
	if err == nil {
		return pm.(*fleet.KVM), err
	}
	return nil, err
}

// BatchUpdateKVMs updates kvms in datastore.
//
// This is a non-atomic operation and doesnt check if the object already exists before
// update. Must be used within a Transaction where objects are checked before update.
// Will lead to partial updates if not used in a transaction.
func BatchUpdateKVMs(ctx context.Context, kvms []*fleet.KVM) ([]*fleet.KVM, error) {
	return putAllKVM(ctx, kvms, true)
}

func putAllKVM(ctx context.Context, kvms []*fleet.KVM, update bool) ([]*fleet.KVM, error) {
	protos := make([]proto.Message, len(kvms))
	updateTime := ptypes.TimestampNow()
	for i, kvm := range kvms {
		kvm.UpdateTime = updateTime
		protos[i] = kvm
	}
	_, err := fleetds.PutAll(ctx, protos, newKVMEntity, update)
	if err == nil {
		return kvms, err
	}
	return nil, err
}

// ImportKVMs creates or updates a batch of kvms in datastore.
func ImportKVMs(ctx context.Context, kvms []*fleet.KVM) (*fleetds.OpResults, error) {
	protos := make([]proto.Message, len(kvms))
	utime := ptypes.TimestampNow()
	for i, m := range kvms {
		m.UpdateTime = utime
		protos[i] = m
	}
	return fleetds.Insert(ctx, protos, newKVMEntity, true, true)
}
