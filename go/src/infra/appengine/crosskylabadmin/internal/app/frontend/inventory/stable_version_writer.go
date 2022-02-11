// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"
	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"

	"go.chromium.org/luci/grpc/grpcutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SetSatlabStableVersion is not yet implemented, but will set the stable version entry for satlab devices.
func (is *ServerImpl) SetSatlabStableVersion(ctx context.Context, req *fleet.SetSatlabStableVersionRequest) (_ *fleet.SetSatlabStableVersionResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	return nil, status.Error(codes.Unimplemented, "SetSatlabStableVersion not yet implemented")
}
