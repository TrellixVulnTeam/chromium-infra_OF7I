// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	luciconfig "go.chromium.org/luci/config"

	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/frontend/fake"

	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
)

type testFixture struct {
	T *testing.T
	C context.Context

	Fleet *FleetServerImpl
}

func newTestFixtureWithContext(ctx context.Context, t *testing.T) (testFixture, func()) {
	tf := testFixture{T: t, C: ctx}
	mc := gomock.NewController(t)

	tf.Fleet = &FleetServerImpl{
		cfgInterfaceFactory:       fakeCfgInterfaceFactory,
		machineDBInterfaceFactory: fakeMachineDBInterface,
		importPageSize:            testImportPageSize,
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
	c = config.Use(c, &config.Config{})
	datastore.GetTestable(c).Consistent(true)
	return c
}

var testImportPageSize = 2
var testMachines = []string{"machine1", "machine2", "machine3"}
var testNics = []*crimson.NIC{
	{
		Name:       "drac",
		Machine:    "machine1",
		MacAddress: "machine1-mac1",
		Switch:     "eq017.atl97",
		Switchport: 1,
		Ipv4:       "ip1.1",
	},
	{
		Name:       "eth0",
		Machine:    "machine1",
		MacAddress: "machine1-mac2",
		Switch:     "eq017.atl97",
		Switchport: 2,
		Ipv4:       "ip1.2",
	},
	{
		Name:       "eth0",
		Machine:    "machine2",
		MacAddress: "machine2-mac1",
		Switch:     "eq017.atl97",
		Switchport: 3,
		Ipv4:       "ip2",
	},
	{
		Name:       "eth0",
		Machine:    "machine3",
		MacAddress: "machine3-mac1",
		Switch:     "eq041.atl97",
		Switchport: 1,
		Ipv4:       "ip3",
	},
}

func fakeMachineDBInterface(ctx context.Context, host string) (crimson.CrimsonClient, error) {
	return &fake.CrimsonClient{
		MachineNames: testMachines,
		Nics:         testNics,
	}, nil
}

func fakeCfgInterfaceFactory(ctx context.Context) luciconfig.Interface {
	return &fake.LuciConfigClient{}
}
