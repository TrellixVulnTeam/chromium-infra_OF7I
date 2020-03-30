// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package datastore

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// FleetEntity represents the interface of entity in datastore.
type FleetEntity interface {
	GetProto() (proto.Message, error)
	GetUpdated() time.Time
}

// ExistsFunc checks if a list of fleet entities exist in datastore.
type ExistsFunc func(context.Context, []FleetEntity) ([]bool, error)

// NewFunc creates a new fleet entity.
type NewFunc func(context.Context, proto.Message, time.Time) (FleetEntity, error)

// QueryAllFunc queries all entities for a given table.
type QueryAllFunc func(context.Context) ([]FleetEntity, error)

// Insert inserts the fleet objects.
func Insert(ctx context.Context, es []proto.Message, nf NewFunc, ef ExistsFunc) (*OpResults, error) {
	allRes := make(OpResults, len(es))
	checkEntities := make([]FleetEntity, 0, len(es))
	checkRes := make(OpResults, 0, len(es))
	updated := time.Now().UTC()
	for i, e := range es {
		allRes[i] = &OpResult{
			Data:      e,
			Timestamp: updated,
		}
		entity, err := nf(ctx, e, updated)
		if err != nil {
			allRes[i].LogError(err)
			continue
		}
		checkEntities = append(checkEntities, entity)
		checkRes = append(checkRes, allRes[i])
	}

	f := func(ctx context.Context) error {
		toAddEntities := make([]FleetEntity, 0, len(checkEntities))
		toAddRes := make(OpResults, 0, len(checkEntities))
		exists, err := ef(ctx, checkEntities)
		if err == nil {
			for i, e := range checkEntities {
				if exists[i] {
					checkRes[i].LogError(errors.Reason("Object exists in the datastore").Err())
					continue
				}
				toAddEntities = append(toAddEntities, e)
				toAddRes = append(toAddRes, checkRes[i])
			}
		} else {
			logging.Debugf(ctx, "Failed to check existence: %s", err)
			toAddEntities = checkEntities
			toAddRes = checkRes
		}
		if err := datastore.Put(ctx, toAddEntities); err != nil {
			for i, e := range err.(errors.MultiError) {
				toAddRes[i].LogError(e)
			}
			return err
		}
		return nil
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return &allRes, err
	}
	return &allRes, nil
}

// GetAll returns all entities in table.
func GetAll(ctx context.Context, qf QueryAllFunc) (*OpResults, error) {
	entities, err := qf(ctx)
	if err != nil {
		return nil, err
	}
	res := make(OpResults, len(entities))
	for i, e := range entities {
		pm, err := e.GetProto()
		if err != nil {
			res[i].LogError(err)
		}
		res[i] = &OpResult{
			Data:      pm,
			Timestamp: e.GetUpdated(),
		}
	}
	return &res, nil
}

// OpResult records the result of datastore operations
type OpResult struct {
	// Operations:
	// Get: record the retrieved proto object.
	// Add: record the proto object to be added.
	// Delete: record the proto object to be deleted.
	// Update: record the proto object to be updated.
	Data proto.Message
	Err  error
	// Operations:
	// Get: record when the proto object is last updated to datastore.
	// Add: record when the proto object is added to datastore.
	// Delete: record when the proto object is removed from datastore.
	// Update: record when the proto object is just updated.
	Timestamp time.Time
}

// LogError logs the error for an operation.
func (op *OpResult) LogError(e error) {
	op.Err = e
}

// OpResults is a list of OpResult.
type OpResults []*OpResult

func (rs OpResults) filter(f func(*OpResult) bool) OpResults {
	result := make(OpResults, 0, len(rs))
	for _, r := range rs {
		if f(r) {
			result = append(result, r)
		}
	}
	return result
}

// Passed generates the list of entities passed the operation.
func (rs OpResults) Passed() OpResults {
	return rs.filter(func(result *OpResult) bool {
		return result.Err == nil
	})
}

// Failed generates the list of entities failed the operation.
func (rs OpResults) Failed() OpResults {
	return rs.filter(func(result *OpResult) bool {
		return result.Err != nil
	})
}
