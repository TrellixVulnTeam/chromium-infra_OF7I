// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"
	"errors"
	"fmt"

	"go.chromium.org/luci/gae/service/datastore"
)

// GetActionEntities reads off all action entities from datastore.
// A limit of zero means that no limit is applied.
func GetActionEntities(ctx context.Context, limit int32) ([]*ActionEntity, error) {
	if limit < 0 {
		return nil, fmt.Errorf("limit cannot be negative: %d", limit)
	}
	query := datastore.NewQuery(ActionKind).Limit(limit)
	if query == nil {
		return nil, errors.New("failed to construct query")
	}
	var entities []*ActionEntity
	if err := datastore.GetAll(ctx, query, &entities); err != nil {
		return nil, err
	}
	return entities, nil
}

// PutActionEntities writes action entities to datastore.
func PutActionEntities(ctx context.Context, entities ...*ActionEntity) error {
	return datastore.Put(ctx, entities)
}
