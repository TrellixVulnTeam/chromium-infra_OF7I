// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package datastore

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Error messages for datastore operations
const (
	InvalidPageToken string = "Invalid Page Token."
	AlreadyExists    string = "Entity already exists."
	NotFound         string = "Entity not found."
	InternalError    string = "Internal Server Error."
	CannotDelete     string = "cannot be deleted"
	InvalidArgument  string = "Invalid argument"
)

// FleetEntity represents the interface of entity in datastore.
type FleetEntity interface {
	GetProto() (proto.Message, error)
}

// NewFunc creates a new fleet entity.
type NewFunc func(context.Context, proto.Message) (FleetEntity, error)

// QueryAllFunc queries all entities for a given table.
type QueryAllFunc func(context.Context) ([]FleetEntity, error)

// Exists checks if a list of fleet entities exist in datastore.
func Exists(ctx context.Context, entities []FleetEntity) ([]bool, error) {
	res, err := datastore.Exists(ctx, entities)
	if err != nil {
		return nil, err
	}
	return res.List(0), nil
}

// Put either creates or updates an entity in the datastore
func Put(ctx context.Context, pm proto.Message, nf NewFunc, update bool) (proto.Message, error) {
	entity, err := nf(ctx, pm)
	if err != nil {
		logging.Errorf(ctx, "Failed to marshal new entity: %s", err)
		return nil, status.Errorf(codes.Internal, InternalError)
	}
	f := func(ctx context.Context) error {
		existsResults, err := datastore.Exists(ctx, entity)
		if err == nil {
			if !existsResults.All() && update {
				return status.Errorf(codes.NotFound, NotFound)
			}
			if existsResults.All() && !update {
				return status.Errorf(codes.AlreadyExists, AlreadyExists)
			}
		} else {
			logging.Debugf(ctx, "Failed to check existence: %s", err)
		}
		if err := datastore.Put(ctx, entity); err != nil {
			logging.Errorf(ctx, "Failed to put in datastore: %s", err)
			return status.Errorf(codes.Internal, InternalError)
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, err
	}
	return pm, nil
}

// PutSingle upserts a single entity in the datastore.
//
// If you have a clean intention to create or update an entity, please use Put().
// This function doesn't need to be called in a transaction.
func PutSingle(ctx context.Context, pm proto.Message, nf NewFunc) (proto.Message, error) {
	entity, err := nf(ctx, pm)
	if err != nil {
		logging.Errorf(ctx, "Failed to marshal new entity: %s", err)
		return nil, status.Errorf(codes.Internal, InternalError)
	}
	if err := datastore.Put(ctx, entity); err != nil {
		logging.Errorf(ctx, "Failed to put in datastore: %s", err)
		return nil, status.Errorf(codes.Internal, InternalError)
	}
	return pm, nil
}

// PutAll Upserts entities in the datastore.
// This is a non-atomic operation and doesnt check if the object already exists before insert/update.
// Returns error even if partial insert/updates succeeds.
// Must be used within a Transaction where objects are checked for existence before update/insert.
// Using it in a Transaction will rollback the partial insert/updates and propagate correct error message.
func PutAll(ctx context.Context, pms []proto.Message, nf NewFunc, update bool) ([]proto.Message, error) {
	entities := make([]FleetEntity, 0, len(pms))
	for _, pm := range pms {
		entity, err := nf(ctx, pm)
		if err != nil {
			logging.Errorf(ctx, "Failed to marshal new entity: %s", err)
			return nil, status.Errorf(codes.Internal, InternalError)
		}
		entities = append(entities, entity)
	}
	if err := datastore.Put(ctx, entities); err != nil {
		logging.Errorf(ctx, "Failed to put in datastore: %s", err)
		return nil, status.Errorf(codes.Internal, InternalError)
	}
	return pms, nil
}

// Get retrieves entity from the datastore.
func Get(ctx context.Context, pm proto.Message, nf NewFunc) (proto.Message, error) {
	entity, err := nf(ctx, pm)
	if err != nil {
		logging.Errorf(ctx, "Failed to marshal new entity: %s", err)
		return nil, status.Errorf(codes.Internal, InternalError)
	}
	if err = datastore.Get(ctx, entity); err != nil {
		if datastore.IsErrNoSuchEntity(err) {
			return nil, status.Errorf(codes.NotFound, NotFound)
		}
		logging.Errorf(ctx, "Failed to get entity from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, InternalError)
	}
	pm, perr := entity.GetProto()
	if perr != nil {
		logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
		return nil, status.Errorf(codes.Internal, InternalError)
	}
	return pm, nil
}

// ListQuery constructs a query to list entities with pagination
func ListQuery(ctx context.Context, entityKind string, pageSize int32, pageToken string, filterMap map[string][]interface{}, keysOnly bool) (q *datastore.Query, err error) {
	var cursor datastore.Cursor
	if pageToken != "" {
		cursor, err = datastore.DecodeCursor(ctx, pageToken)
		if err != nil {
			logging.Errorf(ctx, "Failed to DecodeCursor from pageToken: %s", err)
			return nil, status.Errorf(codes.InvalidArgument, "%s: %s", InvalidPageToken, err.Error())
		}
	}
	q = datastore.NewQuery(entityKind).Limit(pageSize).KeysOnly(keysOnly).FirestoreMode(true)
	for field, values := range filterMap {
		for _, id := range values {
			q = q.Eq(field, id)
		}
	}
	if cursor != nil {
		q = q.Start(cursor)
	}
	return q, nil
}

// Delete deletes the entity from the datastore.
func Delete(ctx context.Context, pm proto.Message, nf NewFunc) error {
	entity, err := nf(ctx, pm)
	if err != nil {
		logging.Errorf(ctx, "Failed to marshal new entity: %s", err)
		return status.Errorf(codes.Internal, InternalError)
	}
	// Datastore doesn't throw an error if the record doesn't exist.
	// Check and return err if there is no such entity in the datastore.
	existsResults, err := datastore.Exists(ctx, entity)
	if err == nil {
		if !existsResults.All() {
			return status.Errorf(codes.NotFound, NotFound)
		}
	} else {
		logging.Debugf(ctx, "Failed to check existence: %s", err)
	}
	if err = datastore.Delete(ctx, entity); err != nil {
		logging.Errorf(ctx, "Failed to delete entity from datastore: %s", err)
		return status.Errorf(codes.Internal, InternalError)
	}
	return nil
}

// Insert inserts the fleet objects.
func Insert(ctx context.Context, es []proto.Message, nf NewFunc, update, upsert bool) (*OpResults, error) {
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
		if upsert {
			toAddEntities = checkEntities
			toAddRes = checkRes
		} else {
			exists, err := Exists(ctx, checkEntities)
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

// BatchGet returns all entities in table for given IDs.
func BatchGet(ctx context.Context, es []proto.Message, nf NewFunc) *OpResults {
	// TODO: (eshwarn)return array of Machines
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

// BatchDelete removes the entities from the datastore
//
// This is a non-atomic operation
// Returns error even if partial delete succeeds.
// Must be used within a Transaction so that partial deletes are rolled back.
// Using it in a Transaction will rollback the partial deletes and propagate correct error message.
func BatchDelete(ctx context.Context, es []proto.Message, nf NewFunc) error {
	checkEntities := make([]FleetEntity, 0, len(es))
	for _, e := range es {
		entity, err := nf(ctx, e)
		if err != nil {
			logging.Errorf(ctx, "Failed to marshal new entity: %s", err)
			return status.Errorf(codes.Internal, InternalError)
		}
		checkEntities = append(checkEntities, entity)
	}
	// Datastore doesn't throw an error if the record doesn't exist.
	// Check and return err if there is no such entity in the datastore.
	exists, err := Exists(ctx, checkEntities)
	if err == nil {
		for i, entity := range checkEntities {
			if !exists[i] {
				errorMsg := fmt.Sprintf("Entity not found: %+v", entity)
				logging.Errorf(ctx, errorMsg)
				return status.Errorf(codes.NotFound, errorMsg)
			}
		}
	}
	if err := datastore.Delete(ctx, checkEntities); err != nil {
		logging.Errorf(ctx, "Failed to delete entities from datastore: %s", err)
		return status.Errorf(codes.Internal, InternalError)
	}
	return nil
}

// DeleteAll removes the entities from the datastore
func DeleteAll(ctx context.Context, es []proto.Message, nf NewFunc) *OpResults {
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
	if len(checkEntities) == 0 {
		return &allRes
	}
	// Datastore doesn't throw an error if the record doesn't exist.
	// Check and return err if there is no such entity in the datastore.
	exists, err := Exists(ctx, checkEntities)
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
