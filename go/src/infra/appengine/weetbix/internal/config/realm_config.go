// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"fmt"

	configpb "infra/appengine/weetbix/internal/config/proto"

	"go.chromium.org/luci/server/auth/realms"
)

// Realm returns the configurations of the requested realm.
func Realm(ctx context.Context, global string) (*configpb.RealmConfig, error) {
	project, realm := realms.Split(global)
	pc, err := Project(ctx, project)
	if err != nil {
		return nil, err
	}
	for _, rc := range pc.GetRealms() {
		if rc.Name == realm {
			return rc, nil
		}
	}
	return nil, fmt.Errorf("no config found for realm %s", global)
}
