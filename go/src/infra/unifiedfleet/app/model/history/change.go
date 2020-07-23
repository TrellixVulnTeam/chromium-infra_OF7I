// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package history

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

// ChangeEventKind is the datastore entity kind for entity changes.
const ChangeEventKind string = "ChangeEvent"

// ChangeEventEntity is a datastore entity that tracks a platform.
type ChangeEventEntity struct {
	_kind string `gae:"$kind,ChangeEvent"`
	// Add an auto-increment ID as key for deleting
	ID        int64  `gae:"$id"`
	Name      string `gae:"name"`
	UserEmail string `gae:"user_email"`
	// ufspb.ChangeEvent cannot be directly used as it contains pointer.
	Change []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled Chrome platform.
func (e *ChangeEventEntity) GetProto() (proto.Message, error) {
	var p ufspb.ChangeEvent
	if err := proto.Unmarshal(e.Change, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newChangeEventEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.ChangeEvent)
	if p.GetName() == "" {
		return nil, errors.Reason("Resource name is not specified in change_event").Err()
	}
	change, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal ChromePlatform %s", p).Err()
	}
	return &ChangeEventEntity{
		Name:      p.GetName(),
		UserEmail: p.GetUserEmail(),
		Change:    change,
	}, nil
}

// CreateBatchChangeEvents creates a batch of new change records in datastore.
func CreateBatchChangeEvents(ctx context.Context, changes []*ufspb.ChangeEvent) ([]*ufspb.ChangeEvent, error) {
	protos := make([]proto.Message, len(changes))
	updateTime := ptypes.TimestampNow()
	for i, change := range changes {
		change.UpdateTime = updateTime
		protos[i] = change
	}
	_, err := ufsds.PutAll(ctx, protos, newChangeEventEntity, false)
	if err == nil {
		return changes, err
	}
	return nil, err
}

// QueryChangesByPropertyName queries change event Entity in the datastore
func QueryChangesByPropertyName(ctx context.Context, propertyName, id string) ([]*ufspb.ChangeEvent, error) {
	q := datastore.NewQuery(ChangeEventKind)
	var entities []*ChangeEventEntity
	if err := datastore.GetAll(ctx, q.Eq(propertyName, id), &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if len(entities) == 0 {
		logging.Infof(ctx, "No change events found for the query: %s", id)
		return nil, nil
	}
	changes := make([]*ufspb.ChangeEvent, 0, len(entities))
	for _, entity := range entities {
		pm, perr := entity.GetProto()
		if perr != nil {
			logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
			continue
		}
		changes = append(changes, pm.(*ufspb.ChangeEvent))
	}
	return changes, nil
}

// GetAllChangeEventEntities returns all change events' entities in datastore.
func GetAllChangeEventEntities(ctx context.Context) ([]*ChangeEventEntity, error) {
	var entities []*ChangeEventEntity
	q := datastore.NewQuery(ChangeEventKind)
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	return entities, nil
}

// DeleteChangeEventEntities deletes a batch of change events' entities
func DeleteChangeEventEntities(ctx context.Context, entities []*ChangeEventEntity) error {
	return datastore.Delete(ctx, entities)
}
