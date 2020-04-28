// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package datastore

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// FleetEntity represents the interface of entity in datastore.
type FleetEntity interface {
	GetProto() (proto.Message, error)
}

// NewFunc creates a new fleet entity.
type NewFunc func(context.Context, proto.Message) (FleetEntity, error)

// QueryAllFunc queries all entities for a given table.
type QueryAllFunc func(context.Context) ([]FleetEntity, error)

// FakeAncestorKey returns a fake datastore key
// A query in transaction requires to have Ancestor filter, see
// https://cloud.google.com/appengine/docs/standard/python/datastore/query-restrictions#queries_inside_transactions_must_include_ancestor_filters
func FakeAncestorKey(ctx context.Context, entityName string) *datastore.Key {
	return datastore.MakeKey(ctx, entityName, "key")
}

// exists checks if a list of fleet entities exist in datastore.
func exists(ctx context.Context, entities []FleetEntity) ([]bool, error) {
	res, err := datastore.Exists(ctx, entities)
	if err != nil {
		return nil, err
	}
	return res.List(0), nil
}

// Insert inserts the fleet objects.
func Insert(ctx context.Context, es []proto.Message, nf NewFunc, update bool) (*OpResults, error) {
	allRes := make(OpResults, len(es))
	checkEntities := make([]FleetEntity, 0, len(es))
	checkRes := make(OpResults, 0, len(es))
	for i, e := range es {
		allRes[i] = &OpResult{
			Data: e,
		}
		entity, err := nf(ctx, e)
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
		exists, err := exists(ctx, checkEntities)
		if err == nil {
			for i, e := range checkEntities {
				if !exists[i] && update {
					checkRes[i].LogError(errors.Reason("No such Object in the datastore").Err())
					continue
				}
				if exists[i] && !update {
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
		res[i] = &OpResult{}
		pm, err := e.GetProto()
		if err != nil {
			res[i].LogError(err)
			continue
		}
		res[i].Data = pm
	}
	return &res, nil
}

// GetByID returns all entities in table for given IDs.
func GetByID(ctx context.Context, es []proto.Message, nf NewFunc) *OpResults {
	allRes := make(OpResults, len(es))
	checkRes := make(OpResults, 0, len(es))
	entities := make([]FleetEntity, 0, len(es))
	for i, e := range es {
		allRes[i] = &OpResult{}
		entity, err := nf(ctx, e)
		if err != nil {
			allRes[i].LogError(err)
			continue
		}
		entities = append(entities, entity)
		checkRes = append(checkRes, allRes[i])
	}

	if err := datastore.Get(ctx, entities); err != nil {
		for i, e := range err.(errors.MultiError) {
			if e != nil {
				checkRes[i].LogError(e)
			}
		}
	}

	for i, e := range entities {
		pm, err := e.GetProto()
		if err != nil {
			checkRes[i].LogError(err)
		}
		checkRes[i].Data = pm
	}
	return &allRes
}

// Delete removes the entities from the datastore
func Delete(ctx context.Context, es []proto.Message, nf NewFunc) *OpResults {
	allRes := make(OpResults, len(es))
	checkRes := make(OpResults, 0, len(es))
	checkEntities := make([]FleetEntity, 0, len(es))
	for i, e := range es {
		allRes[i] = &OpResult{
			Data: e,
		}
		entity, err := nf(ctx, e)
		if err != nil {
			allRes[i].LogError(err)
			continue
		}
		checkEntities = append(checkEntities, entity)
		checkRes = append(checkRes, allRes[i])
	}
	// Datastore doesn't throw an error if the record doesn't exist.
	// Check and return err if there is no such entity in the datastore.
	exists, err := exists(ctx, checkEntities)
	if err == nil {
		for i := range checkEntities {
			if !exists[i] {
				checkRes[i].LogError(errors.Reason("Entity not found").Err())
			}
		}
	}
	if err := datastore.Delete(ctx, checkEntities); err != nil {
		for i, e := range err.(errors.MultiError) {
			if e != nil {
				checkRes[i].LogError(e)
			}
		}
	}
	return &allRes
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
