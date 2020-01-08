// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cfg2datastore

import (
	"context"
	"net/http"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/gae/service/datastore"
	authclient "go.chromium.org/luci/auth"
	gitilesapi "go.chromium.org/luci/common/api/gitiles"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/server/auth"
)

// EntityInterface is the interface that entities can sync with cfg file in git
// repo.
type EntityInterface interface {
	SetUpdatedTime(time.Time)
	GetMessagePayload() (proto.Message, error)
}

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

// DownloadCfgProto downloads the config data in proto.Message.
func DownloadCfgProto(ctx context.Context, client gitiles.GitilesClient, project, committish, path string, msg proto.Message) error {
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

	if err := jsonpb.UnmarshalString(content, msg); err != nil {
		return err
	}
	return nil
}

// SyncProtoToDatastore save the input proto messages to datastore as proper
// entities.
func SyncProtoToDatastore(ctx context.Context, cfgs []proto.Message, entityBuider func(proto.Message) (EntityInterface, error)) error {
	// TODO (guocb) remove stale entities.
	logging.Debugf(ctx, "Updating datastore cache with config messages")
	entities := make([]EntityInterface, 0, len(cfgs))
	now := time.Now().UTC()
	var merr errors.MultiError
	for _, cfg := range cfgs {
		if entity, err := entityBuider(cfg); err != nil {
			merr = append(merr, err)
			continue
		} else {
			entity.SetUpdatedTime(now)
			entities = append(entities, entity)
		}
	}

	logging.Debugf(ctx, "Syncing %d records of data", len(entities))
	if err := datastore.Put(ctx, entities); err != nil {
		return err
	}
	if len(merr) > 0 {
		return merr
	}
	return nil
}

// GetCachedCfgByIds gets the cached config from datastore by Ids.
func GetCachedCfgByIds(ctx context.Context, entities []EntityInterface) ([]proto.Message, error) {
	err := datastore.Get(ctx, entities)

	result := make([]proto.Message, len(entities))
	newErr := errors.NewLazyMultiError(len(entities))
	// Copy all errors returned by Get and plus errors happened during
	// unmarshalling.
	for i := range entities {
		if err != nil && err.(errors.MultiError)[i] != nil {
			newErr.Assign(i, errors.Annotate(err.(errors.MultiError)[i], "get cached config data").Err())
		}
		if cfg, err := entities[i].GetMessagePayload(); err != nil {
			newErr.Assign(i, errors.Annotate(err, "unmarshal config data").Err())
		} else {
			result[i] = cfg
		}
	}
	return result, newErr.Get()
}
