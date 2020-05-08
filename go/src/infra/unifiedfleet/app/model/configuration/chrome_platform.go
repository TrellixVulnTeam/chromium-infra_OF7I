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

	fleet "infra/unifiedfleet/api/v1/proto"
	fleetds "infra/unifiedfleet/app/model/datastore"
)

// ChromePlatformKind is the datastore entity kind for chrome platforms.
const ChromePlatformKind string = "ChromePlatform"

// ChromePlatformEntity is a datastore entity that tracks a platform.
type ChromePlatformEntity struct {
	_kind string `gae:"$kind,ChromePlatform"`
	ID    string `gae:"$id"`
	// fleet.ChromePlatform cannot be directly used as it contains pointer.
	Platform []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled Chrome platform.
func (e *ChromePlatformEntity) GetProto() (proto.Message, error) {
	var p fleet.ChromePlatform
	if err := proto.Unmarshal(e.Platform, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newChromePlatformEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.ChromePlatform)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty Chrome Platform ID").Err()
	}
	platform, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal ChromePlatform %s", p).Err()
	}
	return &ChromePlatformEntity{
		ID:       p.GetName(),
		Platform: platform,
	}, nil
}

func queryAll(ctx context.Context) ([]fleetds.FleetEntity, error) {
	var entities []*ChromePlatformEntity
	q := datastore.NewQuery(ChromePlatformKind)
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	fe := make([]fleetds.FleetEntity, len(entities))
	for i, e := range entities {
		fe[i] = e
	}
	return fe, nil
}

// CreateChromePlatform creates a new chromePlatform in datastore.
func CreateChromePlatform(ctx context.Context, chromePlatform *fleet.ChromePlatform) (*fleet.ChromePlatform, error) {
	return putChromePlatform(ctx, chromePlatform, false)
}

// UpdateChromePlatform updates chromePlatform in datastore.
func UpdateChromePlatform(ctx context.Context, chromePlatform *fleet.ChromePlatform) (*fleet.ChromePlatform, error) {
	return putChromePlatform(ctx, chromePlatform, true)
}

// InsertChromePlatforms inserts chrome platforms to datastore.
func InsertChromePlatforms(ctx context.Context, platforms []*fleet.ChromePlatform) (*fleetds.OpResults, error) {
	protos := make([]proto.Message, len(platforms))
	utime := ptypes.TimestampNow()
	for i, p := range platforms {
		p.UpdateTime = utime
		protos[i] = p
	}
	return fleetds.Insert(ctx, protos, newChromePlatformEntity, false, false)
}

// GetAllChromePlatforms returns all platforms in record.
func GetAllChromePlatforms(ctx context.Context) (*fleetds.OpResults, error) {
	return fleetds.GetAll(ctx, queryAll)
}

func putChromePlatform(ctx context.Context, chromePlatform *fleet.ChromePlatform, update bool) (*fleet.ChromePlatform, error) {
	chromePlatform.UpdateTime = ptypes.TimestampNow()
	pm, err := fleetds.Put(ctx, chromePlatform, newChromePlatformEntity, update)
	if err == nil {
		return pm.(*fleet.ChromePlatform), err
	}
	return nil, err
}
