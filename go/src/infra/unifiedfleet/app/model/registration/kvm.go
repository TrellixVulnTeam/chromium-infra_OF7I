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

// KVMKind is the datastore entity kind KVM.
const KVMKind string = "KVM"

// KVMEntity is a datastore entity that tracks KVM.
type KVMEntity struct {
	_kind            string `gae:"$kind,KVM"`
	ID               string `gae:"$id"`
	ChromePlatformID string `gae:"chrome_platform_id"`
	// ufspb.KVM cannot be directly used as it contains pointer.
	KVM []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled KVM.
func (e *KVMEntity) GetProto() (proto.Message, error) {
	var p ufspb.KVM
	if err := proto.Unmarshal(e.KVM, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newKVMEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.KVM)
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
func QueryKVMByPropertyName(ctx context.Context, propertyName, id string, keysOnly bool) ([]*ufspb.KVM, error) {
	q := datastore.NewQuery(KVMKind).KeysOnly(keysOnly).FirestoreMode(true)
	var entities []*KVMEntity
	if err := datastore.GetAll(ctx, q.Eq(propertyName, id), &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if len(entities) == 0 {
		logging.Infof(ctx, "No kvms found for the query: %s", id)
		return nil, nil
	}
	kvms := make([]*ufspb.KVM, 0, len(entities))
	for _, entity := range entities {
		if keysOnly {
			kvm := &ufspb.KVM{
				Name: entity.ID,
			}
			kvms = append(kvms, kvm)
		} else {
			pm, perr := entity.GetProto()
			if perr != nil {
				logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
				continue
			}
			kvms = append(kvms, pm.(*ufspb.KVM))
		}
	}
	return kvms, nil
}

// CreateKVM creates a new KVM in datastore.
func CreateKVM(ctx context.Context, KVM *ufspb.KVM) (*ufspb.KVM, error) {
	return putKVM(ctx, KVM, false)
}

// UpdateKVM updates KVM in datastore.
func UpdateKVM(ctx context.Context, KVM *ufspb.KVM) (*ufspb.KVM, error) {
	return putKVM(ctx, KVM, true)
}

// GetKVM returns KVM for the given id from datastore.
func GetKVM(ctx context.Context, id string) (*ufspb.KVM, error) {
	pm, err := ufsds.Get(ctx, &ufspb.KVM{Name: id}, newKVMEntity)
	if err == nil {
		return pm.(*ufspb.KVM), err
	}
	return nil, err
}

// ListKVMs lists the KVMs
//
// Does a query over KVM entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListKVMs(ctx context.Context, pageSize int32, pageToken string) (res []*ufspb.KVM, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, KVMKind, pageSize, pageToken, nil, false)
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
		res = append(res, pm.(*ufspb.KVM))
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
		return nil, "", status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteKVM deletes the KVM in datastore
func DeleteKVM(ctx context.Context, id string) error {
	return ufsds.Delete(ctx, &ufspb.KVM{Name: id}, newKVMEntity)
}

func putKVM(ctx context.Context, KVM *ufspb.KVM, update bool) (*ufspb.KVM, error) {
	KVM.UpdateTime = ptypes.TimestampNow()
	pm, err := ufsds.Put(ctx, KVM, newKVMEntity, update)
	if err == nil {
		return pm.(*ufspb.KVM), err
	}
	return nil, err
}

// BatchUpdateKVMs updates kvms in datastore.
//
// This is a non-atomic operation and doesnt check if the object already exists before
// update. Must be used within a Transaction where objects are checked before update.
// Will lead to partial updates if not used in a transaction.
func BatchUpdateKVMs(ctx context.Context, kvms []*ufspb.KVM) ([]*ufspb.KVM, error) {
	return putAllKVM(ctx, kvms, true)
}

func putAllKVM(ctx context.Context, kvms []*ufspb.KVM, update bool) ([]*ufspb.KVM, error) {
	protos := make([]proto.Message, len(kvms))
	updateTime := ptypes.TimestampNow()
	for i, kvm := range kvms {
		kvm.UpdateTime = updateTime
		protos[i] = kvm
	}
	_, err := ufsds.PutAll(ctx, protos, newKVMEntity, update)
	if err == nil {
		return kvms, err
	}
	return nil, err
}

// ImportKVMs creates or updates a batch of kvms in datastore.
func ImportKVMs(ctx context.Context, kvms []*ufspb.KVM) (*ufsds.OpResults, error) {
	protos := make([]proto.Message, len(kvms))
	utime := ptypes.TimestampNow()
	for i, m := range kvms {
		m.UpdateTime = utime
		protos[i] = m
	}
	return ufsds.Insert(ctx, protos, newKVMEntity, true, true)
}

func queryAllKVM(ctx context.Context) ([]ufsds.FleetEntity, error) {
	var entities []*KVMEntity
	q := datastore.NewQuery(KVMKind)
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	fe := make([]ufsds.FleetEntity, len(entities))
	for i, e := range entities {
		fe[i] = e
	}
	return fe, nil
}

// GetAllKVMs returns all kvms in datastore.
func GetAllKVMs(ctx context.Context) (*ufsds.OpResults, error) {
	return ufsds.GetAll(ctx, queryAllKVM)
}

// DeleteKVMs deletes a batch of kvms
func DeleteKVMs(ctx context.Context, resourceNames []string) *ufsds.OpResults {
	protos := make([]proto.Message, len(resourceNames))
	for i, m := range resourceNames {
		protos[i] = &ufspb.KVM{
			Name: m,
		}
	}
	return ufsds.DeleteAll(ctx, protos, newKVMEntity)
}

// BatchDeleteKVMs deletes kvms in datastore.
//
// This is a non-atomic operation. Must be used within a transaction.
// Will lead to partial deletes if not used in a transaction.
func BatchDeleteKVMs(ctx context.Context, ids []string) error {
	protos := make([]proto.Message, len(ids))
	for i, id := range ids {
		protos[i] = &ufspb.KVM{Name: id}
	}
	return ufsds.BatchDelete(ctx, protos, newKVMEntity)
}

// GetKVMIndexedFieldName returns the index name
func GetKVMIndexedFieldName(input string) (string, error) {
	var field string
	input = strings.TrimSpace(input)
	switch strings.ToLower(input) {
	case util.ChromePlatformFilterName:
		field = "chrome_platform_id"
	case util.LabFilterName:
		field = "lab"
	default:
		return "", status.Errorf(codes.InvalidArgument, "Invalid field name %s - field name for KVM are lab/platform", input)
	}
	return field, nil
}
