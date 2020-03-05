// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package bq implements bigquery-related logic.
package bq

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/bq"

	apibq "infra/appengine/cros/lab_inventory/api/bigquery"
	"infra/libs/cros/lab_inventory/datastore"
)

// GetPSTTimeStamp returns the PST timestamp for bq table.
func GetPSTTimeStamp(t time.Time) string {
	tz, _ := time.LoadLocation("America/Los_Angeles")
	return t.In(tz).Format("20060102")
}

// InitBQUploader initialize a bigquery uploader.
func InitBQUploader(ctx context.Context, project, dataset, table string) (*bq.Uploader, error) {
	client, err := bigquery.NewClient(ctx, project)
	if err != nil {
		return nil, err
	}
	up := bq.NewUploader(ctx, client, dataset, table)
	up.SkipInvalidRows = true
	up.IgnoreUnknownValues = true
	return up, nil
}

// GetRegisteredAssetsProtos prepares the proto messages for registered assets to upload to bq.
func GetRegisteredAssetsProtos(ctx context.Context) []proto.Message {
	assets, err := datastore.GetAllAssets(ctx)
	if err != nil {
		return nil
	}
	msgs := make([]proto.Message, len(assets))
	for i, a := range assets {
		msgs[i] = &apibq.RegisteredAsset{
			Id:    a.GetId(),
			Asset: a,
		}
	}
	return msgs
}
