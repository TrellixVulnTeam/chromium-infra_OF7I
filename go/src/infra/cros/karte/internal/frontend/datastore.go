// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	kartepb "infra/cros/karte/api"
	kbqpb "infra/cros/karte/api/bigquery"
	"infra/cros/karte/internal/datastore"
	"infra/cros/karte/internal/errors"
	"infra/cros/karte/internal/filterexp"
	"infra/cros/karte/internal/idserialize"
	"infra/cros/karte/internal/scalars"
)

// defaultBatchSize is the default size of a batch for a datastore query.
const defaultBatchSize = 1000

// ActionKind is the kind of an action
const ActionKind = "ActionKind"

// ObservationKind is the kind of an observation.
const ObservationKind = "ObservationKind"

// ActionEntity is the datastore entity for actions.
//
// Remember to check the setActionEntityFields function.
type ActionEntity struct {
	_kind          string    `gae:"$kind,ActionKind"`
	ID             string    `gae:"$id"`
	Kind           string    `gae:"kind"`
	SwarmingTaskID string    `gae:"swarming_task_id"`
	BuildbucketID  string    `gae:"buildbucket_id"`
	AssetTag       string    `gae:"asset_tag"`
	StartTime      time.Time `gae:"start_time"`
	StopTime       time.Time `gae:"stop_time"`
	CreateTime     time.Time `gae:"receive_time"`
	Status         int32     `gae:"status"`
	FailReason     string    `gae:"fail_reason"`
	SealTime       time.Time `gae:"seal_time"` // After the seal time has passed, no further modifications may be made.
	Hostname       string    `gae:"hostname"`
	// Count the number of times that an action entity was modified by a request.
	ModificationCount int32 `gae:"modification_count"`
	// Deprecated fields!
	ErrorReason string `gae:"error_reason"` // succeeded by "fail_reason'.
}

// ConvertToAction converts a datastore action entity to an action proto.
func (e *ActionEntity) ConvertToAction() *kartepb.Action {
	if e == nil {
		return nil
	}
	return &kartepb.Action{
		Name:              e.ID,
		Kind:              e.Kind,
		SwarmingTaskId:    e.SwarmingTaskID,
		BuildbucketId:     e.BuildbucketID,
		AssetTag:          e.AssetTag,
		StartTime:         scalars.ConvertTimeToTimestampPtr(e.StartTime),
		StopTime:          scalars.ConvertTimeToTimestampPtr(e.StopTime),
		CreateTime:        scalars.ConvertTimeToTimestampPtr(e.CreateTime),
		Status:            scalars.ConvertInt32ToActionStatus(e.Status),
		FailReason:        e.FailReason,
		SealTime:          scalars.ConvertTimeToTimestampPtr(e.SealTime),
		Hostname:          e.Hostname,
		ModificationCount: e.ModificationCount,
	}
}

// ConvertToBQAction converts a datastore action entity to a bigquery proto.
func (e *ActionEntity) ConvertToBQAction() *kbqpb.Action {
	if e == nil {
		return nil
	}
	return &kbqpb.Action{
		Name:           e.ID,
		Kind:           e.Kind,
		SwarmingTaskId: e.SwarmingTaskID,
		BuildbucketId:  e.BuildbucketID,
		AssetTag:       e.AssetTag,
		StartTime:      scalars.ConvertTimeToTimestampPtr(e.StartTime),
		StopTime:       scalars.ConvertTimeToTimestampPtr(e.StopTime),
		CreateTime:     scalars.ConvertTimeToTimestampPtr(e.CreateTime),
		Status:         scalars.ConvertActionStatusIntToString(e.Status),
		FailReason:     e.FailReason,
		SealTime:       scalars.ConvertTimeToTimestampPtr(e.SealTime),
		Hostname:       e.Hostname,
		// ModificationCount is intentionally absent from BigQuery table.
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

// cmp compares two ObservationEntities. ObservationEntities are linearly ordered by all their fields.
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
		return status.Errorf(codes.Internal, "datastore.go Validate: observation entity can have at most one value")
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

// ActionQueryAncillaryData returns ancillary data computed as part of advancing through
// an action entities query.
//
// Currently, we return the biggest (earliest) and smallest (latest) version seen.
type ActionQueryAncillaryData struct {
	BiggestVersion  string
	SmallestVersion string
}

// minVersion computes the minimum of two Karte version strings lexicographically.
func minVersion(a string, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if a <= b {
		return a
	}
	return b
}

// maxVersion computes the maximum of two Karte version strings lexicographically.
func maxVersion(a string, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if a <= b {
		return b
	}
	return a
}

// Next takes a batch size and returns the next batch of action entities from a query.
func (q *ActionEntitiesQuery) Next(ctx context.Context, batchSize int32) ([]*ActionEntity, ActionQueryAncillaryData, error) {
	var d ActionQueryAncillaryData
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
			return nil, ActionQueryAncillaryData{}, errors.Annotate(err, "next action entity: decoding cursor").Err()
		}
		rootedQuery = q.Query.Start(cursor)
	}
	rootedQuery = rootedQuery.Limit(batchSize)
	var entities []*ActionEntity
	err := datastore.Run(ctx, rootedQuery, func(ent *ActionEntity, cb datastore.CursorCB) error {
		// Record the ancillary info! What versions did we see?
		version := idserialize.GetIDVersion(ent.ID)
		d.SmallestVersion = minVersion(d.SmallestVersion, version)
		d.BiggestVersion = maxVersion(d.BiggestVersion, version)
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
	logging.Infof(ctx, "Version range for batch %v", d)
	if err != nil {
		return nil, d, errors.Annotate(err, "next action entity: after running query").Err()
	}
	q.Token = nextToken
	return entities, d, nil
}

// newActionEntitiesQuery makes an action entities query that starts at the position implied
// by the given token and lists all action entities matching the condition described in the
// filter.
func newActionEntitiesQuery(token string, filter string) (*ActionEntitiesQuery, error) {
	expr, err := filterexp.Parse(filter)
	if err != nil {
		// TODO(gregorynisbet): Pick more consistent strategy for assigning error statuses.
		return nil, status.Errorf(codes.InvalidArgument, "make action entities query: %s", err)
	}
	q, err := filterexp.ApplyConditions(
		datastore.NewQuery(ActionKind),
		expr,
	)
	if err != nil {
		return nil, errors.Annotate(err, "make action entities query").Err()
	}
	return &ActionEntitiesQuery{
		Token: token,
		Query: q,
	}, nil
}

// newActionNameRangeQuery takes a beginning name and an end name and produces a query.
//
// This query will apply to names strictly in the range [begin, end).
func newActionNameRangeQuery(begin idserialize.IDInfo, end idserialize.IDInfo) (*ActionEntitiesQuery, error) {
	q := datastore.NewQuery(ActionKind)
	// TODO(gregorynisbet): We can't have multiple inequality constraints in datastore, therefore we filter
	//                      based on the receive time alone and ignore the name (which is based on the time).
	//
	// In the future, consider changing this strategy to take the start and end versions (say "zzzz" and "zzzx" for concreteness)
	// And have this produce multiple queries (one for "zzzz", one for "zzzy", and one for "zzzx").
	bTime := begin.Time()
	eTime := end.Time()
	// The datastore query will actually reject invalid arguments on its own, but we can give the user
	// a better error message if we check the arguments ourselves.
	switch {
	case bTime.After(eTime):
		return nil, errors.Reason("begin time %v is after end time %v", bTime, eTime).Err()
	case bTime.Equal(eTime):
		return nil, errors.Reason("rejecting likely erroneous call: begin time %q and end time are equal %q", bTime.String(), eTime.String()).Err()
	}
	q = q.Gte("receive_time", bTime).Lt("receive_time", eTime)
	return &ActionEntitiesQuery{
		Query: q,
		Token: "",
	}, nil
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

// newObservationEntitiesQuery makes an action entities query that starts at the position
// implied by the page token and lists all action entities.
func newObservationEntitiesQuery(token string, filter string) (*ObservationEntitiesQuery, error) {
	expr, err := filterexp.Parse(filter)
	if err != nil {
		return nil, errors.Annotate(err, "make observation entities query").Err()
	}
	q, err := filterexp.ApplyConditions(
		datastore.NewQuery(ObservationKind),
		expr,
	)
	if err != nil {
		return nil, errors.Annotate(err, "make observation entities query").Err()
	}
	return &ObservationEntitiesQuery{
		Token: token,
		Query: q,
	}, nil
}

// convertActionToActionEntity takes an action and converts it to an action entity.
func convertActionToActionEntity(action *kartepb.Action) (*ActionEntity, error) {
	if action == nil {
		return nil, status.Errorf(codes.Internal, "convert action to action entity: action is nil")
	}
	return &ActionEntity{
		ID:             action.GetName(),
		Kind:           action.Kind,
		SwarmingTaskID: action.SwarmingTaskId,
		BuildbucketID:  action.BuildbucketId,
		AssetTag:       action.AssetTag,
		StartTime:      scalars.ConvertTimestampPtrToTime(action.StartTime),
		StopTime:       scalars.ConvertTimestampPtrToTime(action.StopTime),
		CreateTime:     scalars.ConvertTimestampPtrToTime(action.CreateTime),
		Status:         scalars.ConvertActionStatusToInt32(action.Status),
		FailReason:     action.FailReason,
		SealTime:       scalars.ConvertTimestampPtrToTime(action.SealTime),
		Hostname:       action.Hostname,
	}, nil
}

// PutActionEntities writes action entities to datastore.
func PutActionEntities(ctx context.Context, entities ...*ActionEntity) error {
	// The autogenerated ID should be a string, not an integer.
	// If the value for the ID field is "", an integer value will be
	// autogenerated behind the scenes.
	for _, entity := range entities {
		if entity.ID == "" {
			return fmt.Errorf("put action: entity ID not set or empty")
		}
	}
	return datastore.Put(ctx, entities)
}

// GetActionEntityByID gets an action entity by its ID. If we confirm the absence of an entity successfully, no error is returned.
func GetActionEntityByID(ctx context.Context, id string) (*ActionEntity, error) {
	actionEntity := &ActionEntity{ID: id}
	if err := datastore.Get(ctx, actionEntity); err != nil {
		return nil, errors.Annotate(err, "get action entity by id").Err()
	}
	return actionEntity, nil
}

// convertObservationToObservationEntity takes an observation and converts it to an observation entity.
func convertObservationToObservationEntity(observation *kartepb.Observation) (*ObservationEntity, error) {
	if observation == nil {
		return nil, status.Errorf(codes.Internal, "convert observation to observation entity: action is nil")
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
	// The autogenerated ID should be a string, not an integer.
	// If the value for the ID field is "", an integer value will be
	// autogenerated behind the scenes.
	for _, entity := range entities {
		if entity.ID == "" {
			return fmt.Errorf("put action: entity with empty ID")
		}
	}
	return datastore.Put(ctx, entities)
}

// Copy field values from the right to the left if the field is present in string.
// If fields is empty, copy all fields that are eligible for copying.
// Unrecognized fields are silently ignored.
// Neither left nor right can be nil or else the behavior of this function is undefined.
//
// Keep this function up to date with ActionEntity.
func setActionEntityFields(fields []string, src *ActionEntity, dst *ActionEntity) {
	if src == nil || dst == nil {
		return
	}

	addAll := len(fields) == 0
	m := make(map[string]bool)
	for _, field := range fields {
		m[field] = true
	}

	// Name cannot be copied.
	if addAll || m["kind"] {
		dst.Kind = src.Kind
	}
	if addAll || m["swarming_task_id"] {
		dst.SwarmingTaskID = src.SwarmingTaskID
	}
	if addAll || m["BuildbucketID"] {
		dst.BuildbucketID = src.BuildbucketID
	}
	if addAll || m["asset_tag"] {
		dst.AssetTag = src.AssetTag
	}
	if addAll || m["start_time"] {
		dst.StartTime = src.StartTime
	}
	if addAll || m["stop_time"] {
		dst.StopTime = src.StopTime
	}
	// CreateTime is managed by Karte internally and thus ineligible for copying.
	if addAll || m["status"] {
		dst.Status = src.Status
	}
	if addAll || m["fail_reason"] {
		dst.FailReason = src.FailReason
	}
	// ModificationCount is managed by Karte internally and thus ineligible for copying.
	if addAll || m["error_reason"] {
		dst.ErrorReason = src.ErrorReason
	}
}

// UpdateActionEntity updates an entity according to the field mask.
func UpdateActionEntity(ctx context.Context, entity *ActionEntity, fieldMask []string, increment bool) (*ActionEntity, error) {
	if entity == nil {
		// TODO(gregorynisbet): Remove call to status.Errorf. See b/200578943 for details.
		return nil, status.Errorf(codes.InvalidArgument, "entity cannot be nil")
	}
	if entity.ID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "entity ID cannot be zero")
	}

	// Read the current entity as fullEntity, modify the fields in it that are indicated by fieldMask, and
	// then insert it back into datastore.
	fullEntity, err := GetActionEntityByID(ctx, entity.ID)
	if err != nil {
		logging.Errorf(ctx, "update action entity: datastore error: %s", err)
		return nil, status.Errorf(codes.Aborted, "update action entity: datastore err: %s", err)
	}

	sealTime := fullEntity.SealTime

	if !sealTime.IsZero() && clock.Now(ctx).After(sealTime) {
		return nil, errors.Reason("update action entity: entry sealed at %v", sealTime).Err()
	}

	setActionEntityFields(fieldMask, entity /*src*/, fullEntity /*dst*/)
	// If we're supposed to increment the tally during this update, then increment the tally.
	if increment {
		if fullEntity.ModificationCount < math.MaxInt32 {
			fullEntity.ModificationCount++
		}
	}
	if err := PutActionEntities(ctx, fullEntity); err != nil {
		return nil, errors.Annotate(err, "update action entity").Err()
	}
	return fullEntity, nil
}
