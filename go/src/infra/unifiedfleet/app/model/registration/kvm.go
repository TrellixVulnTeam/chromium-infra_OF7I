// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fleet "infra/unifiedfleet/api/v1/proto"
	fleetds "infra/unifiedfleet/app/model/datastore"
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
	kvms := make([]*fleet.KVM, len(entities))
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

// BatchUpdateKVMs updates kvms in datastore.
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
