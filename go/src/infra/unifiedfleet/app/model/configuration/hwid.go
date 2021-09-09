// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/models"
)

// HwidDataKind is the datastore entity kind HwidData.
const HwidDataKind string = "HwidData"

// HwidDataEntity is a datastore entity that tracks a HwidData.
type HwidDataEntity struct {
	_kind    string `gae:"$kind,HwidData"`
	ID       string `gae:"$id"`
	HwidData []byte `gae:",noindex"`
	Updated  time.Time
}

// GetProto returns the unmarshaled HwidData.
func (e *HwidDataEntity) GetProto() (proto.Message, error) {
	p := &ufspb.DutLabel{}
	if err := proto.Unmarshal(e.HwidData, p); err != nil {
		return nil, err
	}
	return p, nil
}

// UpdateHwidData updates HwidData in datastore.
func UpdateHwidData(ctx context.Context, d *ufspb.DutLabel, hwid string) (*HwidDataEntity, error) {
	hwidData, err := proto.Marshal(d)
	if err != nil {
		return nil, errors.Annotate(err, "failed to marshal HwidData %s", d).Err()
	}

	if hwid == "" {
		return nil, status.Errorf(codes.Internal, "Empty hwid")
	}

	entity := &HwidDataEntity{
		ID:       hwid,
		HwidData: hwidData,
		Updated:  time.Now().UTC(),
	}
	if err := datastore.Put(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

// GetHwidData returns HwidData for the given hwid from datastore.
func GetHwidData(ctx context.Context, hwid string) (*HwidDataEntity, error) {
	entity := &HwidDataEntity{
		ID: hwid,
	}

	if err := datastore.Get(ctx, entity); err != nil {
		if datastore.IsErrNoSuchEntity(err) {
			errorMsg := fmt.Sprintf("Entity not found %+v", entity)
			return nil, status.Errorf(codes.NotFound, errorMsg)
		}
		return nil, err
	}
	return entity, nil
}
