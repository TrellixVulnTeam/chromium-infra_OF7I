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
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/bq"

	apibq "infra/appengine/cros/lab_inventory/api/bigquery"
	"infra/libs/cros/lab_inventory/datastore"
	"infra/libs/cros/lab_inventory/deviceconfig"
	"infra/libs/cros/lab_inventory/manufacturingconfig"
)

// GetPSTTimeStamp returns the PST timestamp for bq table.
func GetPSTTimeStamp(t time.Time) string {
	tz, _ := time.LoadLocation("America/Los_Angeles")
	return t.In(tz).Format("20060102")
}

// InitBQUploaderWithClient initialize a bigquery uploader with a given bigquery client.
func InitBQUploaderWithClient(ctx context.Context, client *bigquery.Client, dataset, table string) *bq.Uploader {
	up := bq.NewUploader(ctx, client, dataset, table)
	up.SkipInvalidRows = true
	up.IgnoreUnknownValues = true
	return up
}

// InitBQUploader initialize a bigquery uploader.
func InitBQUploader(ctx context.Context, project, dataset, table string) (*bq.Uploader, error) {
	client, err := bigquery.NewClient(ctx, project)
	if err != nil {
		return nil, err
	}
	return InitBQUploaderWithClient(ctx, client, dataset, table), nil
}

// GetRegisteredAssetsProtos prepares the proto messages for registered assets to upload to bq.
func GetRegisteredAssetsProtos(ctx context.Context) []proto.Message {
	assets, err := datastore.GetAllAssets(ctx, false)
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

// GetDeviceConfigProtos prepares the proto messages for all device configs to upload to bq.
func GetDeviceConfigProtos(ctx context.Context) []proto.Message {
	devConfigs, err := deviceconfig.GetAllCachedConfig(ctx)
	if err != nil {
		return nil
	}
	msgs := make([]proto.Message, len(devConfigs))
	i := 0
	for dc, t := range devConfigs {
		ut, _ := ptypes.TimestampProto(t)
		msgs[i] = &apibq.DeviceConfigInventory{
			Id:          deviceconfig.GetDeviceConfigIDStr(dc.GetId()),
			Config:      dc,
			UpdatedTime: ut,
		}
		i++
	}
	return msgs
}

// GetManufacturingConfigProtos prepares the proto messages for all device configs to upload to bq.
func GetManufacturingConfigProtos(ctx context.Context) []proto.Message {
	manuConfigs, err := manufacturingconfig.GetAllCachedConfig(ctx)
	if err != nil {
		return nil
	}
	msgs := make([]proto.Message, len(manuConfigs))
	i := 0
	for dc, t := range manuConfigs {
		ut, _ := ptypes.TimestampProto(t)
		msgs[i] = &apibq.ManufacturingInventory{
			ManufacturingId: dc.GetManufacturingId().GetValue(),
			Config:          dc,
			UpdatedTime:     ut,
		}
		i++
	}
	return msgs
}
