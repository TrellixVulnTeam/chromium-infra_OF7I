// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/auth/identity"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"
	"go.chromium.org/luci/server/auth/realms"
	"go.chromium.org/luci/server/auth/service/protocol"
)

func TestCheckPermission(t *testing.T) {
	t.Parallel()

	realmID := "project:some-project"
	admin := identity.Identity("user:admin@example.com")
	reader := identity.Identity("user:reader@example.com")
	writer := identity.Identity("user:writer@example.com")
	readPermission := realms.RegisterPermission("testing.resource.read")
	writePermission := realms.RegisterPermission("testing.resource.write")
	fakeDB := authtest.NewFakeDB(
		authtest.MockMembership(admin, "admins"),
		authtest.MockMembership(reader, "readers"),
		authtest.MockMembership(writer, "writers"),
		authtest.MockPermission(admin, realmID, readPermission),
		authtest.MockPermission(admin, realmID, writePermission),
		authtest.MockPermission(reader, realmID, readPermission),
		authtest.MockPermission(writer, realmID, writePermission),
		authtest.MockRealmData(realmID, &protocol.RealmData{}),
	)
	check := func(id identity.Identity, permission realms.Permission, realm string, expected bool) {
		ctx := auth.WithState(context.Background(), &authtest.FakeState{
			Identity: id,
			FakeDB:   fakeDB,
		})
		err := CheckPermission(ctx, permission, realm)
		if expected {
			So(err, ShouldBeNil)
		} else {
			So(err, ShouldNotBeNil)
		}
	}
	Convey("TestCheckPermission - Read/Write permission check admin", t, func() {
		check(admin, readPermission, realmID, true)
		check(admin, writePermission, realmID, true)
	})
	Convey("TestCheckPermission - Read only permission check for reader", t, func() {
		check(reader, readPermission, realmID, true)
		check(reader, writePermission, realmID, false)
	})
	Convey("TestCheckPermission - Write only permission check for writer", t, func() {
		check(writer, readPermission, realmID, false)
		check(writer, writePermission, realmID, true)
	})
	Convey("TestCheckPermission - Empty realm", t, func() {
		check(writer, readPermission, "", true)
		check(writer, writePermission, "", true)
	})
}
