// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"go.chromium.org/luci/common/logging"
	rpb "infra/appengine/rotation-proxy/proto"
)

// RotationProxyServer implements the proto service RotationProxyService.
type RotationProxyServer struct{}

// BatchGetRotations returns rotation information for a request.
func (rps *RotationProxyServer) BatchGetRotations(ctx context.Context, request *rpb.BatchGetRotationsRequest) (*rpb.BatchGetRotationsResponse, error) {
	logging.Infof(ctx, "Batch Get Rotations")
	return &rpb.BatchGetRotationsResponse{}, nil
}

// BatchUpdateRotations updates rotation information in Rotation Proxy.
func (rps *RotationProxyServer) BatchUpdateRotations(ctx context.Context, request *rpb.BatchUpdateRotationsRequest) (*rpb.BatchUpdateRotationsResponse, error) {
	logging.Infof(ctx, "Batch Update Rotations")
	return &rpb.BatchUpdateRotationsResponse{}, nil
}
