// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/server/auth/realms"

	configpb "infra/appengine/weetbix/internal/config/proto"
)

// RealmNotExistsErr is returned if no configuration could be found
// for the specified realm.
var RealmNotExistsErr = errors.New("no config found for realm")

// Realm returns the configurations of the requested realm.
// If no configuration can be found for the realm, returns
// RealmNotExistsErr.
func Realm(ctx context.Context, global string) (*configpb.RealmConfig, error) {
	project, realm := realms.Split(global)
	pc, err := Project(ctx, project)
	if err != nil {
		if err == NotExistsErr {
			return nil, RealmNotExistsErr
		}
		return nil, err
	}
	for _, rc := range pc.GetRealms() {
		if rc.Name == realm {
			return rc, nil
		}
	}
	return nil, RealmNotExistsErr
}
