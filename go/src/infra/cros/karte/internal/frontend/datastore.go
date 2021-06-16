// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"
	kartepb "infra/cros/karte/api"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
)

// DefaultBatchSize is the default size of a batch for a datastore query.
const defaultBatchSize = 1000

// ActionKind is the kind of an action
const ActionKind = "ActionKind"

// ObservationKind is the kind of an observation.
const ObservationKind = "ObservationKind"

// ActionEntity is the datastore entitiy for actions.
type ActionEntity struct {
	_kind string `gae:"$kind,ActionKind"`
	ID    string `gae:"$id"`
}

// ConvertToAction converts a datastore action entity to an action proto.
func (e *ActionEntity) ConvertToAction() *kartepb.Action {
	return &kartepb.Action{
		Name: e.ID,
	}
}

// ObservationEntity is the datastore entity for observations.
type ObservationEntity struct {
	_kind string `gae:"$kind,ObservationKind"`
	ID    string `gae:"$id"`
}

// ConvertToObservation converts a datastore observation entity to an observation proto.
func (e *ObservationEntity) ConvertToObservation() *kartepb.Observation {
	return &kartepb.Observation{}
}

// ActionEntitiesQuery is a wrapped query of action entities bearing a page token.
type ActionEntitiesQuery struct {
	// Token is the pagination token used by datastore.
	Token string
	// Query is a wrapped datastore query.
	Query *datastore.Query
}

// Next takes a batch size and returns the next batch of action entities from a query.
func (q *ActionEntitiesQuery) Next(ctx context.Context, batchSize int32) ([]*ActionEntity, error) {
	// TODO(gregorynisbet): Consider rejecting defaulted batch sizes instead of
	// applying a default.
	if batchSize == 0 {
		batchSize = defaultBatchSize
		logging.Debugf(ctx, "applied default batch size %d\n", defaultBatchSize)
	}
	var nextToken string
	// A rootedQuery is rooted at the position implied by the pagination token.
	rootedQuery := q.Query
	if q.Token != "" {
		cursor, err := datastore.DecodeCursor(ctx, q.Token)
		if err != nil {
			return nil, err
		}
		rootedQuery = q.Query.Start(cursor)
	}
	rootedQuery = rootedQuery.Limit(batchSize)
	var entities []*ActionEntity
	err := datastore.Run(ctx, rootedQuery, func(ent *ActionEntity, cb datastore.CursorCB) error {
		entities = append(entities, ent)
		// This inequality is weak because this block must run on the last iteration
		// when the query is successful.
		// If the query stops early, we can assume that we have reached the end of the result set
		// and therefore the response token should be empty.
		if len(entities) >= int(batchSize) {
			tok, err := cb()
			if err != nil {
				return err
			}
			nextToken = tok.String()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	q.Token = nextToken
	return entities, nil
}

// MakeAllActionEntitiesQuery makes an action entities query that starts at the position
// implied by the given token and lists all action entities.
func MakeAllActionEntitiesQuery(token string) *ActionEntitiesQuery {
	return &ActionEntitiesQuery{
		Token: token,
		Query: datastore.NewQuery(ActionKind),
	}
}

// ObservationEntitiesQuery is a wrapped query of action entities bearing a page token.
type ObservationEntitiesQuery struct {
	// Token is the pagination token used by datastore.
	Token string
	// Query is a wrapped datastore query.
	Query *datastore.Query
}

// Next takes a batch size and returns the next batch of observation entities from a query.
func (q *ObservationEntitiesQuery) Next(ctx context.Context, batchSize int32) ([]*ObservationEntity, error) {
	// TODO(gregorynisbet): Consider rejecting defaulted batch sizes instead of
	// applying a default.
	if batchSize == 0 {
		batchSize = defaultBatchSize
		logging.Debugf(ctx, "applied default batch size %d\n", defaultBatchSize)
	}
	var nextToken string
	// A rootedQuery is rooted at the position implied by the pagination token.
	rootedQuery := q.Query
	if q.Token != "" {
		cursor, err := datastore.DecodeCursor(ctx, q.Token)
		if err != nil {
			return nil, err
		}
		rootedQuery = q.Query.Start(cursor)
	}
	rootedQuery = rootedQuery.Limit(batchSize)
	var entities []*ObservationEntity
	err := datastore.Run(ctx, rootedQuery, func(ent *ObservationEntity, cb datastore.CursorCB) error {
		entities = append(entities, ent)
		// This inequality is weak because this block must run on the last iteration
		// when the query is successful.
		// If the query stops early, we can assume that we have reached the end of the result set
		// and therefore the response token should be empty.
		if len(entities) >= int(batchSize) {
			tok, err := cb()
			if err != nil {
				return err
			}
			nextToken = tok.String()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	q.Token = nextToken
	return entities, nil
}

// MakeAllObservationEntitiesQuery makes an action entities query that starts at the position
// implied by the page token and lists all action entities.
func MakeAllObservationEntitiesQuery(token string) *ObservationEntitiesQuery {
	return &ObservationEntitiesQuery{
		Token: token,
		Query: datastore.NewQuery(ObservationKind),
	}
}

// ConvertActionToActionEntity takes an action and converts it to an action entity.
func ConvertActionToActionEntity(action *kartepb.Action) (*ActionEntity, error) {
	if action == nil {
		return nil, errors.New("action cannot be nil")
	}
	return &ActionEntity{
		ID: action.GetName(),
	}, nil
}

// PutActionEntities writes action entities to datastore.
func PutActionEntities(ctx context.Context, entities ...*ActionEntity) error {
	return datastore.Put(ctx, entities)
}

// PutObservationEntities writes multiple observation entites to datastore.
func PutObservationEntities(ctx context.Context, entities ...*ObservationEntity) error {
	return datastore.Put(ctx, entities)
}
