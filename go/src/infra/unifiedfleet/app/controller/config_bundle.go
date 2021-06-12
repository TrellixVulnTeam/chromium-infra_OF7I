// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	"infra/unifiedfleet/app/model/configuration"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/config/go/payload"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/genproto/protobuf/field_mask"
)

// UpdateConfigBundle updates ConfigBundle in datastore.
func UpdateConfigBundle(ctx context.Context, cb []byte, mask *field_mask.FieldMask, allowMissing bool) ([]byte, error) {
	// TODO (b/186663540): Implement field mask for partial updates.
	if allowMissing == false {
		logging.Infof(ctx, "UpdateConfigBundle: default to true to ensure upsert of any missing ConfigBundles")
	}
	if mask != nil {
		logging.Infof(ctx, "UpdateConfigBundle: partial updates are not currently supported; the ConfigBundle will be fully updated")
	}

	p := &payload.ConfigBundle{}
	if err := proto.Unmarshal(cb, p); err != nil {
		return nil, err
	}

	id, err := configuration.GenerateCBEntityId(p)
	if err != nil {
		logging.Errorf(ctx, "UpdateConfigBundle failed to generate ConfigBundle entity id: %s", err)
		return nil, err
	}

	if _, err := configuration.UpdateConfigBundle(ctx, p); err != nil {
		return nil, errors.Annotate(err, "UpdateConfigBundle unable to update ConfigBundle: %s", id).Err()
	}

	configData, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "UpdateConfigBundle failed to marshal ConfigBundle: %s", p).Err()
	}

	return configData, nil
}
