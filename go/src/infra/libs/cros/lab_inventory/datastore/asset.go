// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package datastore contains datastore-related logic.
package datastore

import (
	"context"
	"fmt"
	"time"

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/libs/fleet/protos"
)

// AssetOpResult is for use in Datastore to RPC conversions
type AssetOpResult struct {
	Asset       *fleet.ChopsAsset
	Entity      *AssetEntity
	StateEntity *AssetStateEntity
	Err         error
}

func (device *AssetOpResult) logError(e error) {
	device.Err = e
}

// AssetOpResults is a list of AssetOpResult.
type AssetOpResults []AssetOpResult

// Passed generates the list of devices passed the operation.
func (rs AssetOpResults) Passed() []AssetOpResult {
	result := make([]AssetOpResult, 0, len(rs))
	for _, r := range rs {
		if r.Err == nil {
			result = append(result, r)
		}
	}
	return result
}

// ToAsset converts AssetOpResult (format used for datastore) to ChopsAsset (RPC format)
func (device *AssetOpResult) ToAsset() *fleet.ChopsAsset {
	if device.Entity != nil {
		a, err := device.Entity.ToChopsAsset()
		if err != nil {
			fmt.Printf("fail to convert to chopsAsset: %s\n", err.Error())
		}
		return a
	}
	return nil
}

// AddAssets creates a new Asset datastore entity
func AddAssets(ctx context.Context, assets []*fleet.ChopsAsset) ([]*AssetOpResult, error) {
	return putAssets(ctx, assets, false)
}

// UpdateAssets changes the location of the asset
func UpdateAssets(ctx context.Context, assets []*fleet.ChopsAsset) ([]*AssetOpResult, error) {
	return putAssets(ctx, assets, true)
}

// GetAssetsByID returns the asset(s) matching the AssetID
func GetAssetsByID(ctx context.Context, ids []string) []*AssetOpResult {
	queryResults := make([]*AssetOpResult, len(ids))
	entities := make([]AssetEntity, len(ids))
	parent := fakeAncestorKey(ctx)
	for i, assetID := range ids {
		res := &AssetOpResult{
			Entity: &entities[i],
		}
		queryResults[i] = res
		entities[i].ID = assetID
		entities[i].Parent = parent
	}
	if err := datastore.Get(ctx, entities); err != nil {
		if len(ids) > 1 {
			for i, e := range err.(errors.MultiError) {
				queryResults[i].logError(e)
			}
		} else {
			queryResults[0].logError(err)
		}
	}
	return queryResults
}

// GetAssetStatesByID returns the asset(s) matching the AssetID
func GetAssetStatesByID(ctx context.Context, ids []string) []*AssetOpResult {
	queryResults := make([]*AssetOpResult, len(ids))
	entities := make([]AssetStateEntity, len(ids))
	parent := fakeStateAncestorKey(ctx)
	for i, assetID := range ids {
		res := &AssetOpResult{
			StateEntity: &entities[i],
		}
		queryResults[i] = res
		entities[i].ID = assetID
		entities[i].Parent = parent
	}
	if err := datastore.Get(ctx, entities); err != nil {
		if len(ids) > 1 {
			for i, e := range err.(errors.MultiError) {
				queryResults[i].logError(e)
			}
		} else {
			queryResults[0].logError(err)
		}
	}
	return queryResults
}

// DeleteAsset removes the asset from the database
func DeleteAsset(ctx context.Context, ids []string) []*AssetOpResult {
	deleteAssets := make([]*AssetOpResult, len(ids))
	entities := make([]*AssetEntity, len(ids))
	stateEntities := make([]AssetStateEntity, len(ids))
	parent := fakeAncestorKey(ctx)
	stateParent := fakeStateAncestorKey(ctx)
	for i, id := range ids {
		entities[i] = &AssetEntity{
			ID:     id,
			Parent: parent,
		}
		stateEntities[i].ID = id
		stateEntities[i].Parent = stateParent
		req := &AssetOpResult{
			Entity:      entities[i],
			StateEntity: &stateEntities[i],
		}
		deleteAssets[i] = req
	}
	// Datastore doesn't throw an error if the record doesn't exist.
	// Check and return err if there is no such asset in the DB.
	m, err := assetRecordsExists(ctx, entities)
	if err == nil {
		for i := range entities {
			if _, ok := m[i]; !ok {
				deleteAssets[i].logError(errors.Reason("Asset not found").Err())
			}
		}
	}

	if err := datastore.Delete(ctx, entities); err != nil {
		for i, e := range err.(errors.MultiError) {
			if e != nil {
				deleteAssets[i].logError(e)
			}
		}
	}
	// Ignore state update failures. There should be an audit job to periodically check the data consistency.
	datastore.Delete(ctx, stateEntities)
	return deleteAssets
}

// putAssets is used to insert objects in to the datastore. The function
// datastore.Put performs upsert operation, which is create a new object if the
// key doesn't exist else update the object with the given key. update input to
// putAssets determines if an update of existing object is being performed or
// a new object is being created and return the responses accordingly
func putAssets(ctx context.Context, assets []*fleet.ChopsAsset, update bool) ([]*AssetOpResult, error) {
	allResponses := make([]*AssetOpResult, len(assets))
	updated := time.Now().UTC()
	putEntities := make([]*AssetEntity, 0, len(assets))
	putResponses := make([]*AssetOpResult, 0, len(assets))
	var err error
	for i, a := range assets {
		res := &AssetOpResult{
			Asset: a,
		}
		allResponses[i] = res
		ae, err := NewAssetEntity(a, fakeAncestorKey(ctx))
		if err != nil {
			res.logError(err)
			continue
		}
		res.Entity = ae
		newStateEntity, _ := NewAssetStateEntity(a, fleet.State_STATE_ONBOARDING, updated, fakeStateAncestorKey(ctx))
		res.StateEntity = newStateEntity

		putEntities = append(putEntities, ae)
		putResponses = append(putResponses, res)
	}

	f := func(ctx context.Context) error {
		finalEntities := make([]*AssetEntity, 0, len(assets))
		finalResponses := make([]*AssetOpResult, 0, len(assets))
		m, err := assetRecordsExists(ctx, putEntities)
		if err == nil {
			for i, pe := range putEntities {
				_, ok := m[i]
				if !ok && update {
					putResponses[i].logError(errors.Reason("No such asset in the database").Err())
					continue
				}
				if ok && !update {
					putResponses[i].logError(errors.Reason("Asset exists in the database").Err())
					continue
				}
				finalEntities = append(finalEntities, pe)
				finalResponses = append(finalResponses, putResponses[i])
			}
		} else {
			finalEntities = putEntities
			finalResponses = putResponses
		}

		if err := datastore.Put(ctx, finalEntities); err != nil {
			for i, e := range err.(errors.MultiError) {
				finalResponses[i].logError(e)
			}
		}
		return nil
	}
	err = datastore.RunInTransaction(ctx, f, nil)
	// Update asset state
	// Ignore state update failures. There should be an audit job to periodically check the data consistency.
	stateEntities := make([]*AssetStateEntity, 0)
	for _, r := range allResponses {
		if r.Err == nil {
			stateEntities = append(stateEntities, r.StateEntity)
		}
	}
	if err := datastore.Put(ctx, stateEntities); err != nil {
		logging.Errorf(ctx, "fail to save state: %s", err)
	}
	return allResponses, err
}

// A query in transaction requires to have Ancestor filter, see
// https://cloud.google.com/appengine/docs/standard/python/datastore/query-restrictions#queries_inside_transactions_must_include_ancestor_filters
func fakeAncestorKey(ctx context.Context) *datastore.Key {
	return datastore.MakeKey(ctx, AssetEntityName, "key")
}

func fakeStateAncestorKey(ctx context.Context) *datastore.Key {
	return datastore.MakeKey(ctx, AssetStateEntityName, "key")
}

// Checks if the Asset record exists in the database
func assetRecordExists(ctx context.Context, entity *AssetEntity) (bool, error) {
	res, err := datastore.Exists(ctx, entity)
	if res != nil {
		return res.Get(0), err
	}
	return false, err
}

// Checks if the Asset records exist in the database
func assetRecordsExists(ctx context.Context, entities []*AssetEntity) (map[int]bool, error) {
	m := make(map[int]bool, 0)
	res, err := datastore.Exists(ctx, entities)
	if res == nil {
		return m, err
	}
	for i, r := range res.List(0) {
		if r {
			m[i] = true
		}
	}
	return m, err
}
