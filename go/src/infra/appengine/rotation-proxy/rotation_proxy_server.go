// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"sort"
	"time"

	ptypes "github.com/golang/protobuf/ptypes"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	rpb "infra/appengine/rotation-proxy/proto"
)

// Rotation is used to store rpb.Rotation in Datastore.
type Rotation struct {
	Name  string       `gae:"$id"`
	Proto rpb.Rotation `gae:"proto,noindex"`
}

// RotationProxyServer implements the proto service RotationProxyService.
type RotationProxyServer struct{}

// GetRotation returns shift information for a single rotation.
func (rps *RotationProxyServer) GetRotation(ctx context.Context, request *rpb.GetRotationRequest) (*rpb.Rotation, error) {
	rotation := &Rotation{Name: request.Name}

	err := datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		return datastore.Get(ctx, rotation)
	}, nil)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, status.Errorf(codes.NotFound, "rotation %q not found: %v", request.Name, err)
		}
		return nil, err
	}

	if err := processShiftsForRotation(ctx, &rotation.Proto); err != nil {
		return nil, err
	}

	return &rotation.Proto, nil
}

// BatchGetRotations returns shift information for multiple rotations.
func (rps *RotationProxyServer) BatchGetRotations(ctx context.Context, request *rpb.BatchGetRotationsRequest) (*rpb.BatchGetRotationsResponse, error) {
	rotations := make([]*Rotation, len(request.Names))
	for i, rotationName := range request.Names {
		rotations[i] = &Rotation{
			Name: rotationName,
		}
	}
	err := datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		return datastore.Get(ctx, rotations)
	}, nil)
	if err != nil {
		// err should be MultiError, according to https://godoc.org/go.chromium.org/gae/service/datastore#Get
		if firstErr := err.(errors.MultiError).First(); firstErr == datastore.ErrNoSuchEntity {
			return nil, status.Errorf(codes.NotFound, "rotation not found: %v", firstErr)
		}
		return nil, err
	}

	for _, rotation := range rotations {
		if err := processShiftsForRotation(ctx, &rotation.Proto); err != nil {
			return nil, err
		}
	}

	// Compose the response
	rots := make([]*rpb.Rotation, len(rotations))
	for i, rotation := range rotations {
		rots[i] = &rotation.Proto
	}
	return &rpb.BatchGetRotationsResponse{
		Rotations: rots,
	}, nil
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

// processShiftsForRotation filters out past shifts and sort shifts based on start time.
func processShiftsForRotation(ctx context.Context, rotation *rpb.Rotation) error {
	now := clock.Now(ctx)
	n := 0
	for _, shift := range rotation.Shifts {
		var shiftEndTime time.Time
		var err error
		if shift.EndTime != nil {
			shiftEndTime, err = ptypes.Timestamp(shift.EndTime)
			if err != nil {
				return err
			}
		}
		if shift.EndTime == nil || shiftEndTime.After(now) {
			rotation.Shifts[n] = shift
			n++
		}
	}
	rotation.Shifts = rotation.Shifts[:n]
	sort.Slice(rotation.Shifts, func(i, j int) bool {
		return rotation.Shifts[i].StartTime.Seconds < rotation.Shifts[j].StartTime.Seconds
	})
	return nil
}
