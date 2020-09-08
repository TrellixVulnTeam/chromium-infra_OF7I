// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"context"

	"go.chromium.org/luci/server/auth"
)

// BrowserLabAdminRealm is the admin realm for browser lab.
const BrowserLabAdminRealm = "chromium:ufs/browser-admin"

// AtlLabAdminRealm is the admin realm for atl lab.
const AtlLabAdminRealm = "chromium:ufs/atl-admin"

// AcsLabAdminRealm is the admin realm for acs lab.
const AcsLabAdminRealm = "chromium:ufs/acs-admin"

// CurrentUser returns the current user
func CurrentUser(ctx context.Context) string {
	return auth.CurrentUser(ctx).Email
}
