// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package manufacturingconfig

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/infra/proto/go/manufacturing"
	"go.chromium.org/luci/common/proto/gitiles"

	"infra/libs/cros/lab_inventory/cfg2datastore"
)

const entityKind = "ManufacturingConfig"

type manufacturingCfgEntity struct {
	_kind   string `gae:"$kind,ManufacturingConfig"`
	ID      string `gae:"$id"`
	Config  []byte `gae:",noindex"`
	Updated time.Time
}

func (e *manufacturingCfgEntity) SetUpdatedTime(t time.Time) {
	e.Updated = t
}

func (e *manufacturingCfgEntity) GetMessagePayload() (proto.Message, error) {
	cfg := manufacturing.Config{}
	err := proto.Unmarshal(e.Config, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func newManufacturingCfgEntity(msg proto.Message) (cfg2datastore.EntityInterface, error) {
	cfgData, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return &manufacturingCfgEntity{
		ID:     msg.(*manufacturing.Config).GetManufacturingId().GetValue(),
		Config: cfgData,
	}, nil
}

// UpdateDatastore updates the datastore cache for all manufacturing config
// data.
func UpdateDatastore(ctx context.Context, client gitiles.GitilesClient, project, committish, path string) error {
	var allCfgs manufacturing.ConfigList
	cfg2datastore.DownloadCfgProto(ctx, client, project, committish, path, &allCfgs)
	cfgs := make([]proto.Message, len(allCfgs.GetValue()))
	for i, c := range allCfgs.GetValue() {
		cfgs[i] = c
	}

	return cfg2datastore.SyncProtoToDatastore(ctx, cfgs, newManufacturingCfgEntity)
}

// GetCachedConfig gets the manufacturing config data from datastore.
func GetCachedConfig(ctx context.Context, cfgIds []*manufacturing.ConfigID) ([]proto.Message, error) {
	entities := make([]cfg2datastore.EntityInterface, len(cfgIds))
	for i, c := range cfgIds {
		entities[i] = &manufacturingCfgEntity{
			ID: c.GetValue(),
		}
	}
	return cfg2datastore.GetCachedCfgByIds(ctx, entities)
}
