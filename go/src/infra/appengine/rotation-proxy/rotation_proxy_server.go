// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/logging"
	ds "infra/appengine/rotation-proxy/datastore"
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
	// TODO(nqmtuan): Figure out how to make the whole thing transactional.
	// Cross-group transactions seem to be not supported somehow.
	// https://source.chromium.org/chromium/infra/infra/+/master:go/src/go.chromium.org/gae/impl/cloud/datastore.go;l=84
	for _, req := range request.Requests {
		rotation := req.Rotation
		err := datastore.RunInTransaction(ctx, func(ctx context.Context) error {
			return ds.AddOrUpdateRotation(ctx, rotation)
		}, nil)

		if err != nil {
			logging.Errorf(ctx, "Got error: %v", err)
			return nil, err
		}
	}

	// Construct the response
	var rotations []*rpb.Rotation
	for _, req := range request.Requests {
		rotations = append(rotations, req.Rotation)
	}
	return &rpb.BatchUpdateRotationsResponse{
		Rotations: rotations,
	}, nil
}
