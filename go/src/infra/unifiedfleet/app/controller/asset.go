// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	"go.chromium.org/luci/common/errors"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/registration"
)

// AssetRegistration registers the given asset to the datastore after validation
func AssetRegistration(ctx context.Context, asset *ufspb.Asset) (*ufspb.Asset, error) {
	if err := validateAsset(ctx, asset); err != nil {
		return nil, err
	}
	return registration.CreateAsset(ctx, asset)
}

func validateAsset(ctx context.Context, asset *ufspb.Asset) error {
	// Check if the rack exists
	if r := asset.GetLocation().GetRack(); r != "" {
		return ResourceExist(ctx, []*Resource{GetRackResource(r)}, nil)
	}
	return errors.Reason("Invalid rack").Err()
}
