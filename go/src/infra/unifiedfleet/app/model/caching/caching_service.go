// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package caching

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

// CachingServiceKind is the datastore entity kind for chrome platforms.
const CachingServiceKind string = "CachingService"

// CSEntity is a datastore entity that tracks a platform.
type CSEntity struct {
	_kind string `gae:"$kind,CachingService"`
	ID    string `gae:"$id"`
	State string `gae:"state"`
	// b/188491698: The field of Subnet has been deprecated. Use 'Subnets'
	// instead.
	Subnet  string   `gae:"subnet"`
	Subnets []string `gae:"subnets"`
	// ufspb.CachingService cannot be directly used as it contains pointer.
	CachingService []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled CachingService.
func (e *CSEntity) GetProto() (proto.Message, error) {
	var p ufspb.CachingService
	if err := proto.Unmarshal(e.CachingService, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newCSEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.CachingService)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty CachingService ID").Err()
	}
	cs, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal CachingService %s", p).Err()
	}
	return &CSEntity{
		ID:             p.GetName(),
		State:          p.GetState().String(),
		Subnets:        p.GetServingSubnets(),
		CachingService: cs,
	}, nil
}

func queryAll(ctx context.Context) ([]ufsds.FleetEntity, error) {
	var entities []*CSEntity
	q := datastore.NewQuery(CachingServiceKind)
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	fe := make([]ufsds.FleetEntity, len(entities))
	for i, e := range entities {
		fe[i] = e
	}
	return fe, nil
}

// CreateCachingService creates a new CachingService in datastore.
func CreateCachingService(ctx context.Context, cs *ufspb.CachingService) (*ufspb.CachingService, error) {
	return putCachingService(ctx, cs, false)
}

// BatchUpdateCachingServices updates CachingServices in datastore.
//
// This is a non-atomic operation and doesnt check if the object already exists before
// update. Must be used within a transaction where objects are checked before update.
// Will lead to partial updates if not used in a transaction.
func BatchUpdateCachingServices(ctx context.Context, cachingServices []*ufspb.CachingService) ([]*ufspb.CachingService, error) {
	protos := make([]proto.Message, len(cachingServices))
	updateTime := ptypes.TimestampNow()
	for i, cs := range cachingServices {
		cs.UpdateTime = updateTime
		protos[i] = cs
	}
	_, err := ufsds.PutAll(ctx, protos, newCSEntity, true)
	if err == nil {
		return cachingServices, err
	}
	return nil, err
}

// GetCachingService returns CachingService for the given name from datastore.
func GetCachingService(ctx context.Context, name string) (*ufspb.CachingService, error) {
	pm, err := ufsds.Get(ctx, &ufspb.CachingService{Name: name}, newCSEntity)
	if err == nil {
		return pm.(*ufspb.CachingService), err
	}
	return nil, err
}

// DeleteCachingService deletes the CachingService in datastore.
func DeleteCachingService(ctx context.Context, name string) error {
	return ufsds.Delete(ctx, &ufspb.CachingService{Name: name}, newCSEntity)
}

// ListCachingServices lists the CachingServices.
//
// Does a query over CachingService entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListCachingServices(ctx context.Context, pageSize int32, pageToken string, filterMap map[string][]interface{}, keysOnly bool) (res []*ufspb.CachingService, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, CachingServiceKind, pageSize, pageToken, filterMap, keysOnly)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *CSEntity, cb datastore.CursorCB) error {
		if keysOnly {
			CachingService := &ufspb.CachingService{
				Name: ent.ID,
			}
			res = append(res, CachingService)
		} else {
			pm, err := ent.GetProto()
			if err != nil {
				logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
				return nil
			}
			res = append(res, pm.(*ufspb.CachingService))
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
		logging.Errorf(ctx, "Failed to list CachingServices %s", err)
		return nil, "", status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// ListAllCachingServices returns all caching services in datastore.
func ListAllCachingServices(ctx context.Context, keysOnly bool) (res []*ufspb.CachingService, err error) {
	var entities []*CSEntity
	q := datastore.NewQuery(CachingServiceKind).KeysOnly(keysOnly).FirestoreMode(true)
	if err = datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	for _, ent := range entities {
		if keysOnly {
			res = append(res, &ufspb.CachingService{
				Name: ent.ID,
			})
		} else {
			pm, err := ent.GetProto()
			if err != nil {
				logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
				return nil, err
			}
			cachingService := pm.(*ufspb.CachingService)
			res = append(res, cachingService)
		}
	}
	return
}

func putCachingService(ctx context.Context, cs *ufspb.CachingService, update bool) (*ufspb.CachingService, error) {
	cs.UpdateTime = ptypes.TimestampNow()
	pm, err := ufsds.Put(ctx, cs, newCSEntity, update)
	if err == nil {
		return pm.(*ufspb.CachingService), err
	}
	return nil, err
}

// GetCachingServiceIndexedFieldName returns the index name.
func GetCachingServiceIndexedFieldName(input string) (string, error) {
	var field string
	input = strings.TrimSpace(input)
	switch strings.ToLower(input) {
	case util.StateFilterName:
		field = "state"
	case util.SubnetsFilterName:
		field = "subnets"
	default:
		return "", status.Errorf(codes.InvalidArgument, "Invalid field name %s - field name for CachingService are state/subnets", input)
	}
	return field, nil
}
