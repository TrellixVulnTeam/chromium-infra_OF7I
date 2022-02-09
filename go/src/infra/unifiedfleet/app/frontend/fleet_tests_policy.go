// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"

	api "infra/unifiedfleet/api/v1/rpc"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/controller"

	"go.chromium.org/luci/common/logging"
)

// CheckFleetTestsPolicy returns whether the given the test parameters are for a valid test.
func (fs *FleetServerImpl) CheckFleetTestsPolicy(ctx context.Context, req *api.CheckFleetTestsPolicyRequest) (response *ufsAPI.CheckFleetTestsPolicyResponse, err error) {
	validTestResponse := &ufsAPI.CheckFleetTestsPolicyResponse{
		IsTestValid: true,
	}

	// Check test parameters
	err = controller.IsValidTest(ctx, req)
	if err != nil {
		logging.Errorf(ctx, "Error validating test parameters : %s", err.Error())
		return nil, err
	}
	return validTestResponse, nil
}
