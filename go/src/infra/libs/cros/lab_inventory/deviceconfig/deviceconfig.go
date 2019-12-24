// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package deviceconfig

import (
	"net/http"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/gae/service/datastore"
	authclient "go.chromium.org/luci/auth"
	gitilesapi "go.chromium.org/luci/common/api/gitiles"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/server/auth"
	"golang.org/x/net/context"
)

const (
	// DefaultGitilesHost is the default gitiles host which host device config
	// data.
	DefaultGitilesHost = "chrome-internal.googlesource.com"

	// DefaultProject is the default device config repo.
	DefaultProject = "chromeos/infra/config"

	// DefaultCommittish is the default commit of the config file to be
	// downloaded.
	DefaultCommittish = "refs/heads/master"

	// DefaultPath is the default path of device config file.
	DefaultPath = "deviceconfig/generated/device_configs.cfg"

	entityKind = "DevConfig"
)

// NewGitilesClient returns a gitiles client to access the device config data.
func NewGitilesClient(ctx context.Context, host string) (gitiles.GitilesClient, error) {
	logging.Debugf(ctx, "Creating a new gitiles client to access %s", host)
	t, err := auth.GetRPCTransport(ctx, auth.AsSelf, auth.WithScopes(authclient.OAuthScopeEmail, gitilesapi.OAuthScope))
	if err != nil {
		return nil, errors.Annotate(err, "new gitiles client as self").Err()
	}
	client, err := gitilesapi.NewRESTClient(&http.Client{Transport: t}, host, true)
	if err != nil {
		return nil, errors.Annotate(err, "new gitiles client as self").Err()
	}
	return client, nil
}

func downloadDeviceConfigCfg(ctx context.Context, client gitiles.GitilesClient, req *gitiles.DownloadFileRequest) (string, error) {
	rsp, err := client.DownloadFile(ctx, req)
	if err != nil {
		return "", err
	}
	if rsp == nil {
		return "", errors.Reason("failed to download device config cfg file from gitiles").Err()
	}
	logging.Debugf(ctx, "Device config data downloaded successfully")
	return rsp.Contents, nil
}

func getDeviceConfigIDStr(cfgid *device.ConfigId) string {
	var platformID, modelID, variantID, brandID string
	if cfgid.GetPlatformId() != nil {
		platformID = cfgid.GetPlatformId().GetValue()
	}
	if cfgid.GetModelId() != nil {
		modelID = cfgid.GetModelId().GetValue()
	}
	if cfgid.GetVariantId() != nil {
		variantID = cfgid.GetVariantId().GetValue()
	}
	if cfgid.GetBrandId() != nil {
		brandID = cfgid.GetBrandId().GetValue()
	}
	return strings.Join([]string{platformID, modelID, variantID, brandID}, ".")

}

// DeviceEntity is a datastore entity that tracks a device.
type devcfgEntity struct {
	_kind     string `gae:"$kind,DevConfig"`
	ID        string `gae:"$id"`
	DevConfig []byte `gae:",noindex"`
	Updated   time.Time
}

func (e *devcfgEntity) set(cfg *device.Config) error {
	cfgData, err := proto.Marshal(cfg)
	if err != nil {
		return err
	}
	e.ID = getDeviceConfigIDStr(cfg.GetId())
	e.DevConfig = cfgData
	return nil
}

func updateDatastoreCache(ctx context.Context, allCfgs *device.AllConfigs) error {
	// TODO (guocb) remove stale entities.
	logging.Debugf(ctx, "Updating device config datastore cache")
	entities := make([]*devcfgEntity, 0, len(allCfgs.Configs))
	now := time.Now().UTC()
	var merr errors.MultiError
	for _, cfg := range allCfgs.GetConfigs() {
		e := new(devcfgEntity)
		if err := e.set(cfg); err != nil {
			merr = append(merr, err)
			continue
		}
		e.Updated = now
		entities = append(entities, e)
	}

	logging.Infof(ctx, "Syncing %d device config data", len(entities))
	if err := datastore.Put(ctx, entities); err != nil {
		return err
	}
	if len(merr) > 0 {
		return merr
	}
	return nil
}

// UpdateDeviceConfigCache updates the datastore cache for all device config
// data.
func UpdateDeviceConfigCache(ctx context.Context, client gitiles.GitilesClient, project, committish, path string) error {
	logging.Debugf(ctx, "Downloading the device config file %s:%s:%s from gitiles repo", project, committish, path)
	req := &gitiles.DownloadFileRequest{
		Project:    project,
		Committish: committish,
		Path:       path,
		Format:     gitiles.DownloadFileRequest_TEXT,
	}
	content, err := downloadDeviceConfigCfg(ctx, client, req)
	if err != nil {
		return err
	}

	var allCfgs device.AllConfigs
	if err := jsonpb.UnmarshalString(content, &allCfgs); err != nil {
		return err
	}

	return updateDatastoreCache(ctx, &allCfgs)
}

// GetCachedDeviceConfig gets the device config data from datastore.
func GetCachedDeviceConfig(ctx context.Context, cfgIds []*device.ConfigId) ([]*device.Config, error) {
	entities := make([]devcfgEntity, len(cfgIds))
	for i, c := range cfgIds {
		entities[i].ID = getDeviceConfigIDStr(c)
	}
	err := datastore.Get(ctx, entities)

	result := make([]*device.Config, len(cfgIds))
	newErr := errors.NewLazyMultiError(len(entities))
	// Copy all errors returned by Get and plus errors happened during
	// unmarshalling.
	for i := range entities {
		if err != nil && err.(errors.MultiError)[i] != nil {
			newErr.Assign(i, errors.Annotate(err.(errors.MultiError)[i], "get cached device config data").Err())
		}
		cfg := device.Config{}
		unmarshalErr := proto.Unmarshal(entities[i].DevConfig, &cfg)
		if unmarshalErr == nil {
			result[i] = &cfg
		} else {
			newErr.Assign(i, errors.Annotate(unmarshalErr, "unmarshal device config data").Err())
		}
	}
	return result, newErr.Get()
}
