// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package datastore takes care of datastore operations for RotationProxy
package datastore

import (
	"context"

	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	rpb "infra/appengine/rotation-proxy/proto"
)

// Rotation is used to store rpb.Rotation in Datastore.
type Rotation struct {
	Name string `gae:"$id"`
}

// Shift is used to store rpb.Shift in Datastore.
type Shift struct {
	Rotation  *datastore.Key `gae:"$parent"`
	ID        int64          `gae:"$id"`
	Oncalls   []rpb.OncallPerson
	StartTime timestamp.Timestamp
	EndTime   timestamp.Timestamp
}

// AddOrUpdateRotation creates or updates a rotation in datastore
func AddOrUpdateRotation(ctx context.Context, rotation *rpb.Rotation) error {
	// Add the rotation.
	rot := &Rotation{
		Name: rotation.Name,
	}
	if err := datastore.Put(ctx, rot); err != nil {
		return errors.Annotate(err, "Could not create rotation %q", rotation.Name).Err()
	}

	// Delete current shifts for the rotations.
	if err := deleteAllShiftsForRotation(ctx, rot); err != nil {
		return err
	}

	if err := addShiftsForRotation(ctx, rot, rotation.Shifts); err != nil {
		return err
	}
	return nil
}

func deleteAllShiftsForRotation(ctx context.Context, rotation *Rotation) error {
	q := datastore.NewQuery("Shift").Ancestor(datastore.KeyForObj(ctx, rotation))
	shifts := []*Shift{}
	if err := datastore.GetAll(ctx, q, &shifts); err != nil {
		return errors.Annotate(err, "Could not get current shifts for rotation %q", rotation.Name).Err()
	}
	logging.Infof(ctx, "There are %d shifts to be deleted", len(shifts))

	if err := datastore.Delete(ctx, shifts); err != nil {
		return errors.Annotate(err, "Could not delete current shifts for rotation %q", rotation.Name).Err()
	}
	return nil
}

func addShiftsForRotation(ctx context.Context, rotation *Rotation, shifts []*rpb.Shift) error {
	shiftsToAdd := []*Shift{}
	for _, shift := range shifts {
		var oncalls []rpb.OncallPerson
		for _, oc := range shift.Oncalls {
			oncalls = append(oncalls, *oc)
		}

		var startTime timestamp.Timestamp
		if shift.StartTime != nil {
			startTime = *shift.StartTime
		}

		var endTime timestamp.Timestamp
		if shift.EndTime != nil {
			endTime = *shift.EndTime
		}

		sh := &Shift{
			Rotation:  datastore.KeyForObj(ctx, rotation),
			Oncalls:   oncalls,
			StartTime: startTime,
			EndTime:   endTime,
		}
		shiftsToAdd = append(shiftsToAdd, sh)
	}
	if err := datastore.Put(ctx, shiftsToAdd); err != nil {
		return errors.Annotate(err, "Could not add shifts for rotation %q", rotation.Name).Err()
	}
	return nil
}
