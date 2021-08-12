// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"

	kartepb "infra/cros/karte/api"
)

// DefaultBatchSize is the default size of a batch for a datastore query.
const defaultBatchSize = 1000

// ActionKind is the kind of an action
const ActionKind = "ActionKind"

// ObservationKind is the kind of an observation.
const ObservationKind = "ObservationKind"

// ActionEntity is the datastore entity for actions.
type ActionEntity struct {
	_kind          string    `gae:"$kind,ActionKind"`
	ID             string    `gae:"$id"`
	Kind           string    `gae:"kind"`
	SwarmingTaskID string    `gae:"swarming_task_id"`
	AssetTag       string    `gae:"asset_tag"`
	StartTime      time.Time `gae:"start_time"`
	StopTime       time.Time `gae:"stop_time"`
	CreateTime     time.Time `gae:"receive_time"`
	Status         int32     `gae:"status"`
	ErrorReason    string    `gae:"error_reason"`
}

// ConvertToAction converts a datastore action entity to an action proto.
func (e *ActionEntity) ConvertToAction() *kartepb.Action {
	return &kartepb.Action{
		Name:           e.ID,
		Kind:           e.Kind,
		SwarmingTaskId: e.SwarmingTaskID,
		AssetTag:       e.AssetTag,
		StartTime:      convertTimeToTimestampPtr(e.StartTime),
		StopTime:       convertTimeToTimestampPtr(e.StopTime),
		CreateTime:     convertTimeToTimestampPtr(e.CreateTime),
		Status:         convertInt32ToActionStatus(e.Status),
	}
}

// ObservationEntity is the datastore entity for observations.
// Only one of value_string or value_number can have a non-default value. If this constraint is not satisfied, then the record is ill-formed.
type ObservationEntity struct {
	_kind       string  `gae:"$kind,ObservationKind"`
	ID          string  `gae:"$id"`
	ActionID    string  `gae:"action_id"`
	MetricKind  string  `gae:"metric_kind"`
	ValueString string  `gae:"value_string"`
	ValueNumber float64 `gae:"value_number"`
}

// Cmp compares two ObservationEntities. ObservationEntities are linearly ordered by all their fields.
// This order is not related to the semantics of an ObservationEntity.
func (e *ObservationEntity) cmp(o *ObservationEntity) int {
	if e._kind > o._kind {
		return +1
	}
	if e._kind < o._kind {
		return -1
	}
	if e.ID > o.ID {
		return +1
	}
	if e.ID < o.ID {
		return -1
	}
	if e.ActionID > o.ActionID {
		return +1
	}
	if e.ActionID < o.ActionID {
		return -1
	}
	if e.MetricKind > o.MetricKind {
		return +1
	}
	if e.MetricKind < o.MetricKind {
		return -1
	}
	if e.ValueString > o.ValueString {
		return +1
	}
	if e.ValueNumber < o.ValueNumber {
		return -1
	}
	return 0
}

// Validate performs shallow validation on an observation entity.
// It enforces the constraint that only one of ValueString or ValueNumber can have a non-zero value.
func (e *ObservationEntity) Validate() error {
	if e.ValueString == "" && e.ValueNumber == 0.0 {
		return errors.New("datastore.go Validate: observation entity can have at most one value")
	}
	return nil
}

// ConvertToObservation converts a datastore observation entity to an observation proto.
// ConvertToObservation does NOT perform validation on the observation entity it is given;
// this function assumes that its receiver is shallowly valid.
func (e *ObservationEntity) ConvertToObservation() *kartepb.Observation {
	obs := &kartepb.Observation{
		Name:       e.ID,
		ActionName: e.ActionID,
		MetricKind: e.MetricKind,
	}
	if e.ValueString != "" {
		obs.Value = &kartepb.Observation_ValueString{
			ValueString: e.ValueString,
		}
	} else {
		obs.Value = &kartepb.Observation_ValueNumber{
			ValueNumber: e.ValueNumber,
		}
	}
	return obs
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
			return nil, errors.Annotate(err, "next action entity").Err()
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
				return errors.Annotate(err, "next action entity (entities: %d)", len(entities)).Err()
			}
			nextToken = tok.String()
		}
		return nil
	})
	if err != nil {
		return nil, errors.Annotate(err, "next action entity").Err()
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
			return nil, errors.Annotate(err, "next observation entity").Err()
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
				return errors.Annotate(err, "next observation entity").Err()
			}
			nextToken = tok.String()
		}
		return nil
	})
	if err != nil {
		return nil, errors.Annotate(err, "next observation entity").Err()
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
		return nil, errors.New("convert action to action entity: action is nil")
	}
	return &ActionEntity{
		ID:             action.GetName(),
		Kind:           action.Kind,
		SwarmingTaskID: action.SwarmingTaskId,
		AssetTag:       action.AssetTag,
		StartTime:      convertTimestampPtrToTime(action.StartTime),
		StopTime:       convertTimestampPtrToTime(action.StopTime),
		CreateTime:     convertTimestampPtrToTime(action.CreateTime),
		Status:         convertActionStatusToInt32(action.Status),
	}, nil
}

// PutActionEntities writes action entities to datastore.
func PutActionEntities(ctx context.Context, entities ...*ActionEntity) error {
	return datastore.Put(ctx, entities)
}

// ConvertObservationToObservationEntity takes an observation and converts it to an observation entity.
func ConvertObservationToObservationEntity(observation *kartepb.Observation) (*ObservationEntity, error) {
	if observation == nil {
		return nil, errors.New("convert observation to observation entity: action is nil")
	}
	return &ObservationEntity{
		ID:          observation.GetName(),
		ActionID:    observation.GetActionName(),
		MetricKind:  observation.GetMetricKind(),
		ValueString: observation.GetValueString(),
		ValueNumber: observation.GetValueNumber(),
	}, nil
}

// PutObservationEntities writes multiple observation entities to datastore.
func PutObservationEntities(ctx context.Context, entities ...*ObservationEntity) error {
	return datastore.Put(ctx, entities)
}
