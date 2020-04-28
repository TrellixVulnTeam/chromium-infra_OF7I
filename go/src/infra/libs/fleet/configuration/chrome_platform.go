// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"

	fleet "infra/appengine/unified-fleet/api/v1/proto"
	fleetds "infra/libs/fleet/datastore"
)

// ChromePlatformKind is the datastore entity kind for chrome platforms.
const ChromePlatformKind string = "ChromePlatform"

// ChromePlatformEntity is a datastore entity that tracks a platform.
type ChromePlatformEntity struct {
	_kind string `gae:"$kind,ChromePlatform"`
	ID    string `gae:"$id"`
	// fleet.ChromePlatform cannot be directly used as it contains pointer.
	Platform []byte `gae:",noindex"`
	// Should be in UTC timezone.
	Updated time.Time
}

// GetProto returns the unmarshaled Chrome platform.
func (e *ChromePlatformEntity) GetProto() (proto.Message, error) {
	var p fleet.ChromePlatform
	if err := proto.Unmarshal(e.Platform, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetUpdated returns the updated time of the entity.
func (e *ChromePlatformEntity) GetUpdated() time.Time {
	return e.Updated
}

func newEntity(ctx context.Context, pm proto.Message, updateTime time.Time) (fleetds.FleetEntity, error) {
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
		Updated:  updateTime,
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

// InsertChromePlatforms inserts chrome platforms to datastore.
func InsertChromePlatforms(ctx context.Context, platforms []*fleet.ChromePlatform) (*fleetds.OpResults, error) {
	protos := make([]proto.Message, len(platforms))
	for i, p := range platforms {
		protos[i] = p
	}
	return fleetds.Insert(ctx, protos, newEntity, false)
}

// GetAllChromePlatforms returns all platforms in record.
func GetAllChromePlatforms(ctx context.Context) (*fleetds.OpResults, error) {
	return fleetds.GetAll(ctx, queryAll)
}
