// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"

	api "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/controller"

	"go.chromium.org/luci/common/logging"
)

// CheckFleetTestsPolicy returns whether the given the test parameters are for a valid test.
func (fs *FleetServerImpl) CheckFleetTestsPolicy(ctx context.Context, req *api.CheckFleetTestsPolicyRequest) (response *api.CheckFleetTestsPolicyResponse, err error) {
	// Check test parameters
	var statusCode api.TestStatus_Code
	err = controller.IsValidTest(ctx, req)
	if err == nil {
		return &api.CheckFleetTestsPolicyResponse{
			TestStatus: &api.TestStatus{
				Code: api.TestStatus_OK,
			},
		}, nil
	}

	logging.Errorf(ctx, "Returning error %s", err.Error())
	switch err.(type) {
	case *controller.InvalidBoardError:
		statusCode = api.TestStatus_NOT_A_PUBLIC_BOARD
	case *controller.InvalidModelError:
		statusCode = api.TestStatus_NOT_A_PUBLIC_MODEL
	case *controller.InvalidTestError:
		statusCode = api.TestStatus_NOT_A_PUBLIC_TEST
	case *controller.InvalidImageError:
		statusCode = api.TestStatus_NOT_A_PUBLIC_IMAGE
	default:
		return nil, err
	}
	return &api.CheckFleetTestsPolicyResponse{
		TestStatus: &api.TestStatus{
			Code:    statusCode,
			Message: err.Error(),
		},
	}, nil
}
