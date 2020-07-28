// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsds "infra/unifiedfleet/app/model/datastore"
)

// RackLSEPrototypeKind is the datastore entity kind for RackLSEPrototypes.
const RackLSEPrototypeKind string = "RackLSEPrototype"

// RackLSEPrototypeEntity is a datastore entity that tracks a platform.
type RackLSEPrototypeEntity struct {
	_kind string `gae:"$kind,RackLSEPrototype"`
	ID    string `gae:"$id"`
	// ufspb.RackLSEPrototype cannot be directly used as it contains pointer.
	RackLSEPrototype []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled RackLSEPrototype.
func (e *RackLSEPrototypeEntity) GetProto() (proto.Message, error) {
	var p ufspb.RackLSEPrototype
	if err := proto.Unmarshal(e.RackLSEPrototype, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newRackLSEPrototypeEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.RackLSEPrototype)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty RackLSEPrototype ID").Err()
	}
	rackLSEPrototype, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal RackLSEPrototype %s", p).Err()
	}
	return &RackLSEPrototypeEntity{
		ID:               p.GetName(),
		RackLSEPrototype: rackLSEPrototype,
	}, nil
}

// CreateRackLSEPrototype creates a new rackLSEPrototype in datastore.
func CreateRackLSEPrototype(ctx context.Context, rackLSEPrototype *ufspb.RackLSEPrototype) (*ufspb.RackLSEPrototype, error) {
	return putRackLSEPrototype(ctx, rackLSEPrototype, false)
}

// UpdateRackLSEPrototype updates rackLSEPrototype in datastore.
func UpdateRackLSEPrototype(ctx context.Context, rackLSEPrototype *ufspb.RackLSEPrototype) (*ufspb.RackLSEPrototype, error) {
	return putRackLSEPrototype(ctx, rackLSEPrototype, true)
}

// GetRackLSEPrototype returns rackLSEPrototype for the given id from datastore.
func GetRackLSEPrototype(ctx context.Context, id string) (*ufspb.RackLSEPrototype, error) {
	pm, err := ufsds.Get(ctx, &ufspb.RackLSEPrototype{Name: id}, newRackLSEPrototypeEntity)
	if err == nil {
		return pm.(*ufspb.RackLSEPrototype), err
	}
	return nil, err
}

// ListRackLSEPrototypes lists the rackLSEPrototypes
//
// Does a query over RackLSEPrototype entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListRackLSEPrototypes(ctx context.Context, pageSize int32, pageToken string, filterMap map[string][]interface{}, keysOnly bool) (res []*ufspb.RackLSEPrototype, nextPageToken string, err error) {
	// Passing -1 for query limit fetches all the entities from the datastore
	q, err := ufsds.ListQuery(ctx, RackLSEPrototypeKind, -1, pageToken, filterMap, keysOnly)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *RackLSEPrototypeEntity, cb datastore.CursorCB) error {
		if keysOnly {
			rackLSEPrototype := &ufspb.RackLSEPrototype{
				Name: ent.ID,
			}
			res = append(res, rackLSEPrototype)
		} else {
			pm, err := ent.GetProto()
			if err != nil {
				logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
				return nil
			}
			res = append(res, pm.(*ufspb.RackLSEPrototype))
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
		logging.Errorf(ctx, "Failed to List RackLSEPrototype %s", err)
		return nil, "", status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteRackLSEPrototype deletes the rackLSEPrototype in datastore
func DeleteRackLSEPrototype(ctx context.Context, id string) error {
	return ufsds.Delete(ctx, &ufspb.RackLSEPrototype{Name: id}, newRackLSEPrototypeEntity)
}

func putRackLSEPrototype(ctx context.Context, rackLSEPrototype *ufspb.RackLSEPrototype, update bool) (*ufspb.RackLSEPrototype, error) {
	rackLSEPrototype.UpdateTime = ptypes.TimestampNow()
	pm, err := ufsds.Put(ctx, rackLSEPrototype, newRackLSEPrototypeEntity, update)
	if err == nil {
		return pm.(*ufspb.RackLSEPrototype), err
	}
	return nil, err
}

// ImportRackLSEPrototypes creates or updates a batch of rack lse prototypes in datastore
func ImportRackLSEPrototypes(ctx context.Context, lps []*ufspb.RackLSEPrototype) (*ufsds.OpResults, error) {
	protos := make([]proto.Message, len(lps))
	utime := ptypes.TimestampNow()
	for i, m := range lps {
		m.UpdateTime = utime
		protos[i] = m
	}
	return ufsds.Insert(ctx, protos, newRackLSEPrototypeEntity, true, true)
}

// GetRackLSEPrototypeIndexedFieldName returns the index name
func GetRackLSEPrototypeIndexedFieldName(input string) (string, error) {
	return "", status.Errorf(codes.InvalidArgument, "Invalid field %s - No fields available for RackLSEPrototype", input)
}
