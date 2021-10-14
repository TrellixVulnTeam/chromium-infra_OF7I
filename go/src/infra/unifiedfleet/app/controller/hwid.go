// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"math/rand"
	"runtime/debug"
	"time"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/cros/hwid"
	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/util"
)

const cacheAge = time.Hour

// GetHwidDataV1 takes an hwid and returns the Sku and Variant in the form of
// HwidData proto. It will try the following in order:
// 1. Query from datastore. If under an hour old, return data.
// 2. If over an hour old or no data in datastore, attempt to query new data
//    from HWID server.
// 3. If HWID server data available, cache into datastore and return data.
// 4. If server fails, return expired datastore data if present. If not, return
//    nil and error.
func GetHwidDataV1(ctx context.Context, c hwid.ClientInterface, hwid string) (data *ufspb.HwidData, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Reason("Recovered from %v\n%s", r, debug.Stack()).Err()
		}
	}()

	hwidEnt, err := configuration.GetHwidData(ctx, hwid)
	if err != nil && !util.IsNotFoundError(err) {
		return nil, err
	}

	cacheExpired := false
	if hwidEnt != nil {
		cacheExpired = time.Now().UTC().After(hwidEnt.Updated.Add(cacheAge))
	}

	hwidServerOk := rand.Float32() < config.Get(ctx).GetHwidServiceTrafficRatio()
	if hwidServerOk && (hwidEnt == nil || cacheExpired) {
		hwidEntNew, err := fetchHwidData(ctx, c, hwid)
		if err != nil {
			logging.Warningf(ctx, "Error fetching HWID server data: %s", err)
		}

		if hwidEntNew != nil {
			hwidEnt = hwidEntNew
		}
	}
	return configuration.ParseHwidDataV1(hwidEnt)
}

// fetchHwidData queries the hwid server with an hwid and stores the results
// into the UFS datastore.
func fetchHwidData(ctx context.Context, c hwid.ClientInterface, hwid string) (*configuration.HwidDataEntity, error) {
	newData, err := c.QueryHwid(ctx, hwid)
	if err != nil {
		return nil, err
	}

	resp, err := configuration.UpdateHwidData(ctx, newData, hwid)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
