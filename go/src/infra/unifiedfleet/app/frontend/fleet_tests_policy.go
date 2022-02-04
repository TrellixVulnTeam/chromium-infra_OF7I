// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"

	api "infra/unifiedfleet/api/v1/rpc"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
)

// CheckFleetTestsPolicy returns whether the given the test parameters are for a valid test.
// TODO - Implement method
func (fs *FleetServerImpl) CheckFleetTestsPolicy(ctx context.Context, req *api.CheckFleetTestsPolicyRequest) (response *ufsAPI.CheckFleetTestsPolicyResponse, err error) {
	return nil, nil
}
