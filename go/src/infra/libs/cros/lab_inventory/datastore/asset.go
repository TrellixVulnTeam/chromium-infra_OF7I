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

	fleet "infra/libs/fleet/protos"
	ufs "infra/libs/fleet/protos/go"
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

// AssetInfoOpRes is return type for AssetInfo related operations
type AssetInfoOpRes struct {
	AssetInfo *ufs.AssetInfo
	Entity    *AssetInfoEntity
	Err       error
}

// AddAssets creates a new Asset datastore entity
func AddAssets(ctx context.Context, assets []*fleet.ChopsAsset) ([]*AssetOpResult, error) {
	return putAssets(ctx, assets, false)
}

// UpdateAssets changes the location of the asset
func UpdateAssets(ctx context.Context, assets []*fleet.ChopsAsset) ([]*AssetOpResult, error) {
	return putAssets(ctx, assets, true)
}

// GetAllAssets returns all assets from datastore.
func GetAllAssets(ctx context.Context) ([]*fleet.ChopsAsset, error) {
	q := datastore.NewQuery(AssetEntityName).Ancestor(fakeAncestorKey(ctx))
	var assetEntities []*AssetEntity
	if err := datastore.GetAll(ctx, q, &assetEntities); err != nil {
		return nil, err
	}
	assets := make([]*fleet.ChopsAsset, 0)
	for _, ae := range assetEntities {
		if a, err := ae.ToChopsAsset(); err == nil {
			assets = append(assets, a)
		}
	}
	return assets, nil
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

// AddAssetInfo adds the AssetInfo from HaRT to datastore.
//
// All inputs [assetInfo] will get a corresponding response in the order of
// inputs on return, if res.Err != nil then that insert operation failed. Does
// not return an error if the AssetInfo entity already exists in the datastore.
// If assetInfo contains more than one instance with same asset tag, Only one
// of them is inserted into the database and the returned AssetInfoOpRes
// corresponds to the same inserted data for both.
func AddAssetInfo(ctx context.Context, assetInfo []*ufs.AssetInfo) []*AssetInfoOpRes {
	aiEntities := make([]*AssetInfoEntity, 0, len(assetInfo))
	res := make([]*AssetInfoOpRes, 0, len(assetInfo))
	r := make(map[string]*AssetInfoOpRes, len(assetInfo))
	for _, a := range assetInfo {
		assetInfoOpRes := &AssetInfoOpRes{
			AssetInfo: a,
		}
		ent, err := NewAssetInfo(a)
		if err != nil {
			assetInfoOpRes.Err = err
		}
		assetInfoOpRes.Entity = ent
		r[a.GetAssetTag()] = assetInfoOpRes
	}
	for _, a := range r {
		if a.Err == nil {
			aiEntities = append(aiEntities, a.Entity)
		}
	}
	if len(aiEntities) > 0 {
		err := datastore.Put(ctx, aiEntities)
		if err != nil {
			if len(aiEntities) > 1 {
				for i, e := range err.(errors.MultiError) {
					r[aiEntities[i].AssetTag].Err = e
				}
			} else {
				for _, a := range aiEntities {
					r[a.AssetTag].Err = err
				}
			}
		}
	}
	for _, a := range assetInfo {
		res = append(res, r[a.GetAssetTag()])
	}
	return res
}

// GetAssetInfo returns the AssetInfo matching the AssetID
func GetAssetInfo(ctx context.Context, ids []string) []*AssetInfoOpRes {
	queryResults := make([]*AssetInfoOpRes, len(ids))
	qrMap := make(map[string]*AssetInfoOpRes)
	entities := make([]*AssetInfoEntity, 0, len(ids))
	for _, assetID := range ids {
		res := &AssetInfoOpRes{
			Entity: &AssetInfoEntity{
				AssetTag: assetID,
			},
		}
		qrMap[assetID] = res
		// TODO(crbug.com/1074114): Check for "" may not be required
		// depending on how the bug is addressed..
		if assetID != "" {
			entities = append(entities, res.Entity)
		} else {
			res.Err = errors.Reason("Not a valid asset tag").Err()
		}
	}
	if err := datastore.Get(ctx, entities); err != nil {
		for i, e := range err.(errors.MultiError) {
			qrMap[entities[i].AssetTag].Err = e
		}
	}
	for i, assetID := range ids {
		queryResults[i] = qrMap[assetID]
	}
	return queryResults
}
