// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package deviceconfig

import (
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/proto/gitiles"
	"golang.org/x/net/context"

	"infra/libs/cros/lab_inventory/cfg2datastore"
)

const entityKind = "DevConfig"

// GetDeviceConfigIDStr returns a string as device config short name.
func GetDeviceConfigIDStr(cfgid *device.ConfigId) string {
	// TODO (guocb) Add `BranchID` as part of DeviceConfigID.
	var platformID, modelID, variantID string
	if v := cfgid.GetPlatformId(); v != nil {
		platformID = strings.ToLower(v.GetValue())
	}
	if v := cfgid.GetModelId(); v != nil {
		modelID = strings.ToLower(v.GetValue())
	}
	if v := cfgid.GetVariantId(); v != nil {
		variantID = strings.ToLower(v.GetValue())
	}
	return strings.Join([]string{platformID, modelID, variantID}, ".")

}

type devcfgEntity struct {
	_kind     string `gae:"$kind,DevConfig"`
	ID        string `gae:"$id"`
	DevConfig []byte `gae:",noindex"`
	Updated   time.Time
}

func (e *devcfgEntity) SetUpdatedTime(t time.Time) {
	e.Updated = t
}

func (e *devcfgEntity) GetMessagePayload() (proto.Message, error) {
	cfg := device.Config{}
	err := proto.Unmarshal(e.DevConfig, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (e *devcfgEntity) GetID() string {
	return e.ID
}

func newDevCfgEntity(msg proto.Message) (cfg2datastore.EntityInterface, error) {
	cfgData, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return &devcfgEntity{
		ID:        GetDeviceConfigIDStr(msg.(*device.Config).GetId()),
		DevConfig: cfgData,
	}, nil
}

// UpdateDatastore updates the datastore cache for all device config data.
func UpdateDatastore(ctx context.Context, client gitiles.GitilesClient, project, committish, path string) error {
	var allCfgs device.AllConfigs
	cfg2datastore.DownloadCfgProto(ctx, client, project, committish, path, &allCfgs)
	cfgs := make([]proto.Message, len(allCfgs.GetConfigs()))
	for i, c := range allCfgs.GetConfigs() {
		cfgs[i] = c
	}

	return cfg2datastore.SyncProtoToDatastore(ctx, cfgs, newDevCfgEntity)
}

// GetCachedConfig gets the device config data from datastore.
func GetCachedConfig(ctx context.Context, cfgIds []*device.ConfigId) ([]proto.Message, error) {
	entities := make([]cfg2datastore.EntityInterface, len(cfgIds))
	for i, c := range cfgIds {
		e := devcfgEntity{
			ID: GetDeviceConfigIDStr(c),
		}
		logging.Debugf(ctx, "Getting devconfig for ID: '%s'", e.ID)
		entities[i] = &e
	}
	return cfg2datastore.GetCachedCfgByIds(ctx, entities)
}

// GetAllCachedConfig gets all the device configs from datastore.
func GetAllCachedConfig(ctx context.Context) (map[*device.Config]time.Time, error) {
	var entities []*devcfgEntity
	if err := datastore.GetAll(ctx, datastore.NewQuery(entityKind), &entities); err != nil {
		return nil, err
	}
	configs := make(map[*device.Config]time.Time, 0)
	for _, dc := range entities {
		if a, err := dc.GetMessagePayload(); err == nil {
			configs[a.(*device.Config)] = dc.Updated
		}
	}
	return configs, nil
}

// DeviceConfigsExists Checks if the device configs exist in the datastore
func DeviceConfigsExists(ctx context.Context, cfgIds []*device.ConfigId) (map[int32]bool, error) {
	entities := make([]cfg2datastore.EntityInterface, len(cfgIds))
	for i, c := range cfgIds {
		e := devcfgEntity{
			ID: GetDeviceConfigIDStr(c),
		}
		logging.Debugf(ctx, "Check devconfig for ID: '%s'", e.ID)
		entities[i] = &e
	}
	res, err := datastore.Exists(ctx, entities)
	if err != nil {
		return nil, err
	}
	m := make(map[int32]bool, 0)
	for i, r := range res.List(0) {
		if r {
			m[int32(i)] = true
		}
	}
	return m, err
}
