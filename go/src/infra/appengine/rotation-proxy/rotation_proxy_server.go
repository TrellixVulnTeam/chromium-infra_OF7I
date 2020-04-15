// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/logging"
	rpb "infra/appengine/rotation-proxy/proto"
)

// Rotation is used to store rpb.Rotation in Datastore.
type Rotation struct {
	Name  string       `gae:"$id"`
	Proto rpb.Rotation `gae:"proto,noindex"`
}

// RotationProxyServer implements the proto service RotationProxyService.
type RotationProxyServer struct{}

// BatchGetRotations returns rotation information for a request.
func (rps *RotationProxyServer) BatchGetRotations(ctx context.Context, request *rpb.BatchGetRotationsRequest) (*rpb.BatchGetRotationsResponse, error) {
	logging.Infof(ctx, "Batch Get Rotations")
	return &rpb.BatchGetRotationsResponse{}, nil
}

// BatchUpdateRotations updates rotation information in Rotation Proxy.
func (rps *RotationProxyServer) BatchUpdateRotations(ctx context.Context, request *rpb.BatchUpdateRotationsRequest) (*rpb.BatchUpdateRotationsResponse, error) {
	entities := make([]*Rotation, len(request.Requests))
	for i, req := range request.Requests {
		entities[i] = &Rotation{
			Name:  req.Rotation.Name,
			Proto: *req.Rotation,
		}
	}
	err := datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		return datastore.Put(ctx, entities)
	}, nil)
	if err != nil {
		return nil, err
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
