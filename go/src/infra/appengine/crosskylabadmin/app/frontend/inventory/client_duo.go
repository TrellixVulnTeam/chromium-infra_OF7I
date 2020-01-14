// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"
	"math/rand"
	"time"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/appengine/crosskylabadmin/app/frontend/internal/gitstore"
	"infra/libs/skylab/inventory"
)

// TODO (guocb) Add deadline to ensure it won't hung.
// TODO (guocb) Recover the workflow in case of panic.
type duoClient struct {
	gc *gitStoreClient
	ic *invServiceClient

	// A number in [0, 100] indicate the traffic duplicated to inventory
	// service.
	trafficRatio int
}

func newDuoClient(ctx context.Context, gs *gitstore.InventoryStore, host string, trafficRatio int) (inventoryClient, error) {
	gc, err := newGitStoreClient(ctx, gs)
	if err != nil {
		return nil, errors.Annotate(err, "create git client").Err()
	}
	ic, err := newInvServiceClient(ctx, host)
	if err != nil {
		logging.Infof(ctx, "Failed to create inventory client of the duo client. Just return the git store client")
		return gc, nil
	}
	return &duoClient{
		gc:           gc.(*gitStoreClient),
		ic:           ic.(*invServiceClient),
		trafficRatio: trafficRatio,
	}, nil
}

func (client *duoClient) willDupToV2() bool {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Intn(100) < client.trafficRatio
}

func (client *duoClient) addManyDUTsToFleet(ctx context.Context, nds []*inventory.CommonDeviceSpecs, pickServoPort bool) (string, []*inventory.CommonDeviceSpecs, error) {
	url, ds, err := client.gc.addManyDUTsToFleet(ctx, nds, pickServoPort)
	logging.Infof(ctx, "[v1] add dut result: %s, %s", url, err)
	logging.Infof(ctx, "[v1] spec returned: %s", ds)

	if client.willDupToV2() {
		url2, ds2, err2 := client.ic.addManyDUTsToFleet(ctx, ds, pickServoPort)
		logging.Infof(ctx, "[v2] add dut result: %s, %s", url2, err2)
		logging.Infof(ctx, "[v2] spec returned: %s", ds2)
	}
	return url, ds, err
}
