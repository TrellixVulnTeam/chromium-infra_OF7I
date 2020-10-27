// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/auth/identity"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"
	"go.chromium.org/luci/server/auth/realms"

	"infra/unifiedfleet/app/config"
)

// error msgs used for testing
const (
	CannotCreate string = "Cannot create"
)

func testingContext() context.Context {
	c := gaetesting.TestingContextWithAppID("dev~infra-unified-fleet-system")
	c = gologger.StdConfig.Use(c)
	c = logging.SetLevel(c, logging.Debug)
	c = config.Use(c, &config.Config{})
	datastore.GetTestable(c).Consistent(true)
	return c
}

func initializeFakeAuthDB(ctx context.Context, id identity.Identity, permission realms.Permission, realm string) context.Context {
	return auth.WithState(ctx, &authtest.FakeState{
		Identity: id,
		FakeDB: authtest.NewFakeDB(
			authtest.MockMembership(id, "user"),
			authtest.MockPermission(id, realm, permission),
		),
	})
}
