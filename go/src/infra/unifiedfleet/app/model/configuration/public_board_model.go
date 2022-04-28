// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PublicBoardModelDataKind is the datastore entity kind PublicBoardModelData.
const PublicBoardModelDataKind string = "PublicBoardModelData"

// PublicBoardModelDataEntity is a datastore entity that tracks a PublicBoardModelData.
type PublicBoardModelDataEntity struct {
	_kind  string   `gae:"$kind,PublicBoardModel"`
	Board  string   `gae:"$id"`
	Models []string `gae:"models"`
}

// AddPublicBoardModelData adds a public board name and its corresponding models in the datastore.
func AddPublicBoardModelData(ctx context.Context, board string, models []string) (*PublicBoardModelDataEntity, error) {
	if board == "" {
		return nil, status.Errorf(codes.Internal, "Empty board")
	}

	entity := &PublicBoardModelDataEntity{
		Board:  board,
		Models: models,
	}
	if err := datastore.Put(ctx, entity); err != nil {
		logging.Errorf(ctx, "Failed to put board name in datastore : %s - %s", board, err)
		return nil, err
	}
	return entity, nil
}

// GetPublicBoardModelData returns PublicBoardModelData for the given board from datastore.
func GetPublicBoardModelData(ctx context.Context, board string) (*PublicBoardModelDataEntity, error) {
	entity := &PublicBoardModelDataEntity{
		Board: board,
	}

	if err := datastore.Get(ctx, entity); err != nil {
		if datastore.IsErrNoSuchEntity(err) {
			errorMsg := fmt.Sprintf("Entity not found %+v", entity)
			return nil, status.Errorf(codes.NotFound, errorMsg)
		}
		return nil, err
	}
	return entity, nil
}
