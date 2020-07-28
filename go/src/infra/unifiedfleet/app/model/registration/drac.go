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

// DracKind is the datastore entity kind Drac.
const DracKind string = "Drac"

// DracEntity is a datastore entity that tdracs Drac.
type DracEntity struct {
	_kind    string `gae:"$kind,Drac"`
	ID       string `gae:"$id"`
	SwitchID string `gae:"switch_id"`
	// ufspb.Drac cannot be directly used as it contains pointer.
	Drac []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled Drac.
func (e *DracEntity) GetProto() (proto.Message, error) {
	var p ufspb.Drac
	if err := proto.Unmarshal(e.Drac, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newDracEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.Drac)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty Drac ID").Err()
	}
	drac, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal Drac %s", p).Err()
	}
	return &DracEntity{
		ID:       p.GetName(),
		SwitchID: p.GetSwitchInterface().GetSwitch(),
		Drac:     drac,
	}, nil
}

// QueryDracByPropertyName query's Drac Entity in the datastore
//
// If keysOnly is true, then only key field is populated in returned dracs
func QueryDracByPropertyName(ctx context.Context, propertyName, id string, keysOnly bool) ([]*ufspb.Drac, error) {
	q := datastore.NewQuery(DracKind).KeysOnly(keysOnly).FirestoreMode(true)
	var entities []*DracEntity
	if err := datastore.GetAll(ctx, q.Eq(propertyName, id), &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if len(entities) == 0 {
		logging.Infof(ctx, "No dracs found for the query: %s", id)
		return nil, nil
	}
	dracs := make([]*ufspb.Drac, 0, len(entities))
	for _, entity := range entities {
		if keysOnly {
			drac := &ufspb.Drac{
				Name: entity.ID,
			}
			dracs = append(dracs, drac)
		} else {
			pm, perr := entity.GetProto()
			if perr != nil {
				logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
				continue
			}
			dracs = append(dracs, pm.(*ufspb.Drac))
		}
	}
	return dracs, nil
}

// CreateDrac creates a new drac in datastore.
func CreateDrac(ctx context.Context, drac *ufspb.Drac) (*ufspb.Drac, error) {
	return putDrac(ctx, drac, false)
}

// UpdateDrac updates drac in datastore.
func UpdateDrac(ctx context.Context, drac *ufspb.Drac) (*ufspb.Drac, error) {
	return putDrac(ctx, drac, true)
}

// GetDrac returns drac for the given id from datastore.
func GetDrac(ctx context.Context, id string) (*ufspb.Drac, error) {
	pm, err := ufsds.Get(ctx, &ufspb.Drac{Name: id}, newDracEntity)
	if err == nil {
		return pm.(*ufspb.Drac), err
	}
	return nil, err
}

// ListDracs lists the dracs
//
// Does a query over Drac entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListDracs(ctx context.Context, pageSize int32, pageToken string, filterMap map[string][]interface{}, keysOnly bool) (res []*ufspb.Drac, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, DracKind, pageSize, pageToken, filterMap, keysOnly)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *DracEntity, cb datastore.CursorCB) error {
		if keysOnly {
			drac := &ufspb.Drac{
				Name: ent.ID,
			}
			res = append(res, drac)
		} else {
			pm, err := ent.GetProto()
			if err != nil {
				logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
				return nil
			}
			res = append(res, pm.(*ufspb.Drac))
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
		logging.Errorf(ctx, "Failed to List Dracs %s", err)
		return nil, "", status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteDrac deletes the drac in datastore
func DeleteDrac(ctx context.Context, id string) error {
	return ufsds.Delete(ctx, &ufspb.Drac{Name: id}, newDracEntity)
}

func putDrac(ctx context.Context, drac *ufspb.Drac, update bool) (*ufspb.Drac, error) {
	drac.UpdateTime = ptypes.TimestampNow()
	pm, err := ufsds.Put(ctx, drac, newDracEntity, update)
	if err == nil {
		return pm.(*ufspb.Drac), err
	}
	return nil, err
}

// BatchUpdateDracs updates dracs in datastore.
//
// This is a non-atomic operation and doesnt check if the object already exists before
// update. Must be used within a Transaction where objects are checked before update.
// Will lead to partial updates if not used in a transaction.
func BatchUpdateDracs(ctx context.Context, dracs []*ufspb.Drac) ([]*ufspb.Drac, error) {
	return putAllDrac(ctx, dracs, true)
}

func putAllDrac(ctx context.Context, dracs []*ufspb.Drac, update bool) ([]*ufspb.Drac, error) {
	protos := make([]proto.Message, len(dracs))
	updateTime := ptypes.TimestampNow()
	for i, drac := range dracs {
		drac.UpdateTime = updateTime
		protos[i] = drac
	}
	_, err := ufsds.PutAll(ctx, protos, newDracEntity, update)
	if err == nil {
		return dracs, err
	}
	return nil, err
}

// ImportDracs creates or updates a batch of dracs in datastore.
func ImportDracs(ctx context.Context, dracs []*ufspb.Drac) (*ufsds.OpResults, error) {
	protos := make([]proto.Message, len(dracs))
	utime := ptypes.TimestampNow()
	for i, m := range dracs {
		m.UpdateTime = utime
		protos[i] = m
	}
	return ufsds.Insert(ctx, protos, newDracEntity, true, true)
}

func queryAllDrac(ctx context.Context) ([]ufsds.FleetEntity, error) {
	var entities []*DracEntity
	q := datastore.NewQuery(DracKind)
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	fe := make([]ufsds.FleetEntity, len(entities))
	for i, e := range entities {
		fe[i] = e
	}
	return fe, nil
}

// GetAllDracs returns all dracs in datastore.
func GetAllDracs(ctx context.Context) (*ufsds.OpResults, error) {
	return ufsds.GetAll(ctx, queryAllDrac)
}

// DeleteDracs deletes a batch of dracs
func DeleteDracs(ctx context.Context, resourceNames []string) *ufsds.OpResults {
	protos := make([]proto.Message, len(resourceNames))
	for i, m := range resourceNames {
		protos[i] = &ufspb.Drac{
			Name: m,
		}
	}
	return ufsds.DeleteAll(ctx, protos, newDracEntity)
}

// GetDracIndexedFieldName returns the index name
func GetDracIndexedFieldName(input string) (string, error) {
	var field string
	input = strings.TrimSpace(input)
	switch strings.ToLower(input) {
	case util.SwitchFilterName:
		field = "switch_id"
	case util.LabFilterName:
		field = "lab"
	case util.RackFilterName:
		field = "rack"
	default:
		return "", status.Errorf(codes.InvalidArgument, "Invalid field name %s - field name for drac are lab/rack/switch", input)
	}
	return field, nil
}
