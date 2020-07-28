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

// RPMKind is the datastore entity kind RPM.
const RPMKind string = "RPM"

// RPMEntity is a datastore entity that tracks RPM.
type RPMEntity struct {
	_kind string `gae:"$kind,RPM"`
	ID    string `gae:"$id"`
	// ufspb.RPM cannot be directly used as it contains pointer.
	RPM []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled RPM.
func (e *RPMEntity) GetProto() (proto.Message, error) {
	var p ufspb.RPM
	if err := proto.Unmarshal(e.RPM, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newRPMEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.RPM)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty RPM ID").Err()
	}
	rpm, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal RPM %s", p).Err()
	}
	return &RPMEntity{
		ID:  p.GetName(),
		RPM: rpm,
	}, nil
}

// CreateRPM creates a new RPM in datastore.
func CreateRPM(ctx context.Context, RPM *ufspb.RPM) (*ufspb.RPM, error) {
	return putRPM(ctx, RPM, false)
}

// UpdateRPM updates RPM in datastore.
func UpdateRPM(ctx context.Context, RPM *ufspb.RPM) (*ufspb.RPM, error) {
	return putRPM(ctx, RPM, true)
}

// GetRPM returns RPM for the given id from datastore.
func GetRPM(ctx context.Context, id string) (*ufspb.RPM, error) {
	pm, err := ufsds.Get(ctx, &ufspb.RPM{Name: id}, newRPMEntity)
	if err == nil {
		return pm.(*ufspb.RPM), err
	}
	return nil, err
}

// ListRPMs lists the RPMs
//
// Does a query over RPM entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListRPMs(ctx context.Context, pageSize int32, pageToken string) (res []*ufspb.RPM, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, RPMKind, pageSize, pageToken, nil, false)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *RPMEntity, cb datastore.CursorCB) error {
		pm, err := ent.GetProto()
		if err != nil {
			logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
			return nil
		}
		res = append(res, pm.(*ufspb.RPM))
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to List RPMs %s", err)
		return nil, "", status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteRPM deletes the RPM in datastore
func DeleteRPM(ctx context.Context, id string) error {
	return ufsds.Delete(ctx, &ufspb.RPM{Name: id}, newRPMEntity)
}

func putRPM(ctx context.Context, RPM *ufspb.RPM, update bool) (*ufspb.RPM, error) {
	RPM.UpdateTime = ptypes.TimestampNow()
	pm, err := ufsds.Put(ctx, RPM, newRPMEntity, update)
	if err == nil {
		return pm.(*ufspb.RPM), err
	}
	return nil, err
}

// BatchDeleteRPMs deletes rpms in datastore.
//
// This is a non-atomic operation. Must be used within a transaction.
// Will lead to partial deletes if not used in a transaction.
func BatchDeleteRPMs(ctx context.Context, ids []string) error {
	protos := make([]proto.Message, len(ids))
	for i, id := range ids {
		protos[i] = &ufspb.RPM{Name: id}
	}
	return ufsds.BatchDelete(ctx, protos, newRPMEntity)
}

// BatchUpdateRPMs updates rpms in datastore.
//
// This is a non-atomic operation and doesnt check if the object already exists before
// update. Must be used within a Transaction where objects are checked before update.
// Will lead to partial updates if not used in a transaction.
func BatchUpdateRPMs(ctx context.Context, rpms []*ufspb.RPM) ([]*ufspb.RPM, error) {
	return putAllRPM(ctx, rpms, true)
}

func putAllRPM(ctx context.Context, rpms []*ufspb.RPM, update bool) ([]*ufspb.RPM, error) {
	protos := make([]proto.Message, len(rpms))
	updateTime := ptypes.TimestampNow()
	for i, rpm := range rpms {
		rpm.UpdateTime = updateTime
		protos[i] = rpm
	}
	_, err := ufsds.PutAll(ctx, protos, newRPMEntity, update)
	if err == nil {
		return rpms, err
	}
	return nil, err
}

// GetRPMIndexedFieldName returns the index name
func GetRPMIndexedFieldName(input string) (string, error) {
	var field string
	input = strings.TrimSpace(input)
	switch strings.ToLower(input) {
	case util.LabFilterName:
		field = "lab"
	default:
		return "", status.Errorf(codes.InvalidArgument, "Invalid field name %s - field name for RPM are lab", input)
	}
	return field, nil
}
