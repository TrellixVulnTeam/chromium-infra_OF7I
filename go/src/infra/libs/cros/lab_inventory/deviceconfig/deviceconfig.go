// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package deviceconfig

import (
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/luci/common/proto/gitiles"
	"golang.org/x/net/context"

	"infra/libs/cros/lab_inventory/cfg2datastore"
)

const entityKind = "DevConfig"

func getDeviceConfigIDStr(cfgid *device.ConfigId) string {
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

func newDevCfgEntity(msg proto.Message) (cfg2datastore.EntityInterface, error) {
	cfgData, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return &devcfgEntity{
		ID:        getDeviceConfigIDStr(msg.(*device.Config).GetId()),
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
		entities[i] = &devcfgEntity{
			ID: getDeviceConfigIDStr(c),
		}
	}
	return cfg2datastore.GetCachedCfgByIds(ctx, entities)
}
