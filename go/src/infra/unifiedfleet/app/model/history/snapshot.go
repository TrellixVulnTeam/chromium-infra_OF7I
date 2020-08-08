// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package history

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SnapshotMsgKind is the datastore entity kind for storing the snapshot msgs of resources.
const SnapshotMsgKind string = "SnapshotMsg"

// SnapshotMsgEntity is a datastore entity that stores the snapshot msgs.
type SnapshotMsgEntity struct {
	_kind string `gae:"$kind,SnapshotMsg"`
	// Add an auto-increment ID as key for deleting
	ID           int64  `gae:"$id"`
	ResourceName string `gae:"resource_name"`
	Delete       bool   `gae:"delete"`
	Msg          []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled Chrome platform.
func (e *SnapshotMsgEntity) GetProto(t proto.Message) error {
	if err := proto.Unmarshal(e.Msg, t); err != nil {
		return err
	}
	return nil
}

// NewSnapshotMsgEntity creates a new SnapshotMsgEntity
func NewSnapshotMsgEntity(resourceName string, delete bool, pm proto.Message) (*SnapshotMsgEntity, error) {
	msg, err := proto.Marshal(pm)
	if err != nil {
		return nil, err
	}
	return &SnapshotMsgEntity{
		ResourceName: resourceName,
		Delete:       delete,
		Msg:          msg,
	}, nil
}

// BatchUpdateSnapshotMsg updates a batch of new snapshot msgs
func BatchUpdateSnapshotMsg(ctx context.Context, msgs []*SnapshotMsgEntity) error {
	if err := datastore.Put(ctx, msgs); err != nil {
		logging.Errorf(ctx, "Failed to put snapshot msg in datastore: %s", err)
		return status.Errorf(codes.Internal, err.Error())
	}
	return nil
}

// GetAllSnapshotMsg returns all snapshot msg entities in datastore.
func GetAllSnapshotMsg(ctx context.Context) ([]*SnapshotMsgEntity, error) {
	var entities []*SnapshotMsgEntity
	q := datastore.NewQuery(SnapshotMsgKind)
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	return entities, nil
}

// DeleteSnapshotMsgEntities deletes a batch of snapshot msg entities
func DeleteSnapshotMsgEntities(ctx context.Context, entities []*SnapshotMsgEntity) error {
	return datastore.Delete(ctx, entities)
}

// QuerySnapshotMsgByPropertyName queries snapshot msg entity in the datastore
func QuerySnapshotMsgByPropertyName(ctx context.Context, propertyName, id string) ([]*SnapshotMsgEntity, error) {
	q := datastore.NewQuery(SnapshotMsgKind)
	var entities []*SnapshotMsgEntity
	if err := datastore.GetAll(ctx, q.Eq(propertyName, id), &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return entities, nil
}
