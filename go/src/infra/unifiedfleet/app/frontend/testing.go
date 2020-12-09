// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/gae/service/datastore"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/external"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/util"
)

type testFixture struct {
	T *testing.T
	C context.Context

	Fleet *FleetServerImpl
}

func newTestFixtureWithContext(ctx context.Context, t *testing.T) (testFixture, func()) {
	tf := testFixture{T: t, C: external.WithTestingContext(ctx)}
	mc := gomock.NewController(t)

	tf.Fleet = &FleetServerImpl{
		importPageSize: testImportPageSize,
	}
	validate := func() {
		mc.Finish()
	}
	return tf, validate
}

func testingContext() context.Context {
	c := gaetesting.TestingContextWithAppID("dev~infra-unified-fleet-system")
	c = gologger.StdConfig.Use(c)
	c = logging.SetLevel(c, logging.Debug)
	c = config.Use(c, &config.Config{
		CrosNetworkConfig: &config.OSNetworkConfig{
			GitilesHost: "test_gitiles",
			Project:     "test_project",
			Branch:      "test_branch",
			CrosNetworkTopology: []*config.OSNetworkConfig_OSNetworkTopology{
				{
					Name:       "test_name",
					RemotePath: "test_git_path",
					SheetId:    "test_sheet",
				},
			},
		},
	})
	c = external.WithTestingContext(c)
	datastore.GetTestable(c).Consistent(true)
	return c
}

var testImportPageSize = 2

func setupTestVlan(ctx context.Context) {
	vlan := &ufspb.Vlan{
		Name:        "vlan-1",
		VlanAddress: "192.168.40.0/22",
	}
	configuration.CreateVlan(ctx, vlan)
	ips, _, _, _, _ := util.ParseVlan(vlan.GetName(), vlan.GetVlanAddress(), vlan.GetFreeStartIpv4Str(), vlan.GetFreeEndIpv4Str())
	// Only import the first 20 as one single transaction cannot import all.
	configuration.ImportIPs(ctx, ips[0:20])
}
