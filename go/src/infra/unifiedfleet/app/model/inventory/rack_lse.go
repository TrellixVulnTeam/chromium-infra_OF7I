// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

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

// RackLSEKind is the datastore entity kind RackLSE.
const RackLSEKind string = "RackLSE"

// RackLSEEntity is a datastore entity that tracks RackLSE.
type RackLSEEntity struct {
	_kind              string   `gae:"$kind,RackLSE"`
	ID                 string   `gae:"$id"`
	RackIDs            []string `gae:"rack_ids"`
	RackLSEProtoTypeID string   `gae:"racklse_prototype_id"`
	KVMIDs             []string `gae:"kvm_ids"`
	RPMIDs             []string `gae:"rpm_ids"`
	SwitchIDs          []string `gae:"switch_ids"`
	// ufspb.RackLSE cannot be directly used as it contains pointer.
	RackLSE []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled RackLSE.
func (e *RackLSEEntity) GetProto() (proto.Message, error) {
	var p ufspb.RackLSE
	if err := proto.Unmarshal(e.RackLSE, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newRackLSEEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.RackLSE)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty RackLSE ID").Err()
	}
	rackLSE, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal RackLSE %s", p).Err()
	}
	return &RackLSEEntity{
		ID:                 p.GetName(),
		RackIDs:            p.GetRacks(),
		RackLSEProtoTypeID: p.GetRackLsePrototype(),
		KVMIDs:             p.GetChromeosRackLse().GetKvms(),
		RPMIDs:             p.GetChromeosRackLse().GetRpms(),
		SwitchIDs:          p.GetChromeosRackLse().GetSwitches(),
		RackLSE:            rackLSE,
	}, nil
}

// QueryRackLSEByPropertyName queries RackLSE Entity in the datastore
//
// If keysOnly is true, then only key field is populated in returned racklses
func QueryRackLSEByPropertyName(ctx context.Context, propertyName, id string, keysOnly bool) ([]*ufspb.RackLSE, error) {
	q := datastore.NewQuery(RackLSEKind).KeysOnly(keysOnly).FirestoreMode(true)
	var entities []*RackLSEEntity
	if err := datastore.GetAll(ctx, q.Eq(propertyName, id), &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if len(entities) == 0 {
		logging.Infof(ctx, "No rackLSEs found for the query: %s", id)
		return nil, nil
	}
	rackLSEs := make([]*ufspb.RackLSE, 0, len(entities))
	for _, entity := range entities {
		if keysOnly {
			rackLSE := &ufspb.RackLSE{
				Name: entity.ID,
			}
			rackLSEs = append(rackLSEs, rackLSE)
		} else {
			pm, perr := entity.GetProto()
			if perr != nil {
				logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
				continue
			}
			rackLSEs = append(rackLSEs, pm.(*ufspb.RackLSE))
		}
	}
	return rackLSEs, nil
}

// CreateRackLSE creates a new rackLSE in datastore.
func CreateRackLSE(ctx context.Context, rackLSE *ufspb.RackLSE) (*ufspb.RackLSE, error) {
	return putRackLSE(ctx, rackLSE, false)
}

// UpdateRackLSE updates rackLSE in datastore.
func UpdateRackLSE(ctx context.Context, rackLSE *ufspb.RackLSE) (*ufspb.RackLSE, error) {
	return putRackLSE(ctx, rackLSE, true)
}

// GetRackLSE returns rack for the given id from datastore.
func GetRackLSE(ctx context.Context, id string) (*ufspb.RackLSE, error) {
	pm, err := ufsds.Get(ctx, &ufspb.RackLSE{Name: id}, newRackLSEEntity)
	if err == nil {
		return pm.(*ufspb.RackLSE), err
	}
	return nil, err
}

// ListRackLSEs lists the racks
//
// Does a query over RackLSE entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListRackLSEs(ctx context.Context, pageSize int32, pageToken string, filterMap map[string][]interface{}, keysOnly bool) (res []*ufspb.RackLSE, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, RackLSEKind, pageSize, pageToken, filterMap, keysOnly)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *RackLSEEntity, cb datastore.CursorCB) error {
		if keysOnly {
			rackLSE := &ufspb.RackLSE{
				Name: ent.ID,
			}
			res = append(res, rackLSE)
		} else {
			pm, err := ent.GetProto()
			if err != nil {
				logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
				return nil
			}
			res = append(res, pm.(*ufspb.RackLSE))
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
		logging.Errorf(ctx, "Failed to List RackLSEs %s", err)
		return nil, "", status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteRackLSE deletes the rackLSE in datastore
func DeleteRackLSE(ctx context.Context, id string) error {
	return ufsds.Delete(ctx, &ufspb.RackLSE{Name: id}, newRackLSEEntity)
}

// BatchUpdateRackLSEs updates rackLSEs in datastore.
//
// This is a non-atomic operation and doesnt check if the object already exists before
// update. Must be used within a Transaction where objects are checked before update.
// Will lead to partial updates if not used in a transaction.
func BatchUpdateRackLSEs(ctx context.Context, rackLSEs []*ufspb.RackLSE) ([]*ufspb.RackLSE, error) {
	return putAllRackLSE(ctx, rackLSEs, true)
}

func putRackLSE(ctx context.Context, rackLSE *ufspb.RackLSE, update bool) (*ufspb.RackLSE, error) {
	rackLSE.UpdateTime = ptypes.TimestampNow()
	pm, err := ufsds.Put(ctx, rackLSE, newRackLSEEntity, update)
	if err == nil {
		return pm.(*ufspb.RackLSE), err
	}
	return nil, err
}

func putAllRackLSE(ctx context.Context, rackLSEs []*ufspb.RackLSE, update bool) ([]*ufspb.RackLSE, error) {
	protos := make([]proto.Message, len(rackLSEs))
	updateTime := ptypes.TimestampNow()
	for i, rackLSE := range rackLSEs {
		rackLSE.UpdateTime = updateTime
		protos[i] = rackLSE
	}
	_, err := ufsds.PutAll(ctx, protos, newRackLSEEntity, update)
	if err == nil {
		return rackLSEs, err
	}
	return nil, err
}

// ImportRackLSEs creates or updates a batch of rack LSEs in datastore
func ImportRackLSEs(ctx context.Context, lses []*ufspb.RackLSE) (*ufsds.OpResults, error) {
	protos := make([]proto.Message, len(lses))
	utime := ptypes.TimestampNow()
	for i, m := range lses {
		m.UpdateTime = utime
		protos[i] = m
	}
	return ufsds.Insert(ctx, protos, newRackLSEEntity, true, true)
}

// GetRackLSEIndexedFieldName returns the index name
func GetRackLSEIndexedFieldName(input string) (string, error) {
	var field string
	input = strings.TrimSpace(input)
	switch strings.ToLower(input) {
	case util.SwitchFilterName:
		field = "switch_ids"
	case util.RPMFilterName:
		field = "rpm_ids"
	case util.KVMFilterName:
		field = "kvm_ids"
	case util.LabFilterName:
		field = "lab"
	case util.RackFilterName:
		field = "rack_ids"
	case util.RackPrototypeFilterName:
		field = "racklse_prototype_id"
	default:
		return "", status.Errorf(codes.InvalidArgument, "Invalid field name %s - field name for racklse are rack/rackprototype/kvm/rpm/switch/lab", input)
	}
	return field, nil
}
