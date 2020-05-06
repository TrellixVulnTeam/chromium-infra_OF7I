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

func fakeMachineDBInterface(ctx context.Context, host string) (crimson.CrimsonClient, error) {
	return &fake.CrimsonClient{
		MachineNames: testMachines,
	}, nil
}
