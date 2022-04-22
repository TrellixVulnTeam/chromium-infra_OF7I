// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package asset

import (
	"context"
	"errors"
	proto "infra/appengine/poros/api/proto"

	"github.com/google/uuid"
	"go.chromium.org/luci/gae/service/datastore"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type AssetHandler struct {
	UnimplementedAssetServer
}

// Creates the given Asset.
func (e *AssetHandler) Create(ctx context.Context, req *CreateAssetRequest) (*proto.AssetEntity, error) {
	// validate name & description
	if req.GetName() == "" || req.GetDescription() == "" {
		return nil, errors.New("name or description cannot be empty")
	}
	id := uuid.New().String()
	currentTime := timestamppb.Now()
	err := datastore.Put(ctx, &proto.AssetEntity{
		AssetId:     id,
		Name:        req.GetName(),
		Description: req.GetDescription(),
		CreatedAt:   currentTime,
	})
	if err != nil {
		return nil, err
	}
	return getById(ctx, id)
}

// Retrieves a Asset for a given unique value.
func (e *AssetHandler) Get(ctx context.Context, req *GetAssetRequest) (*proto.AssetEntity, error) {
	return getById(ctx, req.GetAssetId())
}

// Update a single asset in Enterprise Asset.
func (e *AssetHandler) Update(ctx context.Context, req *UpdateAssetRequest) (*proto.AssetEntity, error) {
	id := req.GetAsset().GetAssetId()
	mask := req.GetUpdateMask()

	if mask == nil || len(mask.GetPaths()) == 0 || !mask.IsValid(req.GetAsset()) {
		return nil, errors.New("Update Mask can't be empty or invalid")
	}
	// In a transaction load asset, set fields based on field mask.
	err := datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		asset := &proto.AssetEntity{AssetId: id}
		if err := datastore.Get(ctx, asset); err != nil {
			return err
		}
		asset.ModifiedAt = timestamppb.Now()
		err := datastore.Put(ctx, id, &asset)
		return err
	}, nil)

	if err != nil {
		return nil, err
	}

	return getById(ctx, id)
}

// Deletes the given Asset.
func (e *AssetHandler) Delete(ctx context.Context, req *DeleteAssetRequest) (*emptypb.Empty, error) {
	err := datastore.Delete(ctx, &proto.AssetEntity{
		AssetId: req.GetAssetId()})
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// Lists all Assets.
func (e *AssetHandler) List(ctx context.Context, in *ListAssetsRequest) (*ListAssetsResponse, error) {
	// TODO: crbug/1318606 - Implement Asset List functionality with filter & paging.
	assets := []proto.AssetEntity{}
	res := &ListAssetsResponse{}
	query := datastore.NewQuery("AssetEntity").Order("created_at")
	err := datastore.GetAll(ctx, query, &assets)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(assets); i++ {
		res.Assets = append(res.Assets, &assets[i])
	}
	return res, nil
}

func getById(ctx context.Context, id string) (*proto.AssetEntity, error) {
	asset := &proto.AssetEntity{AssetId: id}
	if err := datastore.Get(ctx, asset); err != nil {
		return nil, err
	}
	return asset, nil
}
