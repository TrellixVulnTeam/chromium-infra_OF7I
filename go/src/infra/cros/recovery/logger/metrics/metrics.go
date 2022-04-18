// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package metrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"
)

// ActionStatus is the status of an action.
type ActionStatus string

const (
	// ActionStatusUnspecified is an unknown status.
	ActionStatusUnspecified ActionStatus = ""
	// ActionStatusSuccess represents a successful action.
	ActionStatusSuccess ActionStatus = "success"
	// ActionStatusFail represents a failed action.
	ActionStatusFail ActionStatus = "fail"
	// ActionStatusSkip represents a skipped action.
	// TODO(gregorynisbet): Add support for skipped actions to Karte.
	ActionStatusSkip ActionStatus = "skip"
)

// A ValueType is the type of an observation, such as a number or a string.
type ValueType string

const (
	// ValueTypeUnspecified is an unknown value type.
	ValueTypeUnspecified ValueType = ""
	// ValueTypeString represents a string-valued measurement.
	ValueTypeString ValueType = "string"
	// ValueTypeNumber represents a real-valued measurement.
	ValueTypeNumber ValueType = "number"
)

// Action is an event performed on a DUT.
// TODO(gregorynisbet): Rename an action to something else so we don't collide with the other notion of an action.
type Action struct {
	// Name is the identifier for an action. It is controlled by Karte.
	Name string
	// ActionKind is a coarse-grained type of observation e.g. "ssh".
	ActionKind string
	// SwarmingTaskID is the ID of the associated swarming task.
	SwarmingTaskID string
	// BuildbucketID is the ID of the buildbucket build.
	BuildbucketID string
	// AssetTag is the asset tag of the DUT that the observation is recorded for.
	AssetTag string
	// StartTime is when the event started.
	StartTime time.Time
	// StopTime is when the event ended.
	StopTime time.Time
	// Status is whether the event was successful, failed, or unknown.
	Status ActionStatus
	// Hostname is the hostname of the device or the name of the unit.
	Hostname string
	// FailReason is an error message with information describing the failure.
	FailReason string
	// Observations are the observations associated with the current observation.
	Observations []*Observation
}

// Observation is the type of a measurement associated with an event performed on a DUT.
type Observation struct {
	// MetricKind is the metric kind (e.g. battery percentage).
	MetricKind string
	// ValueType is the type of value (e.g. String).
	ValueType ValueType
	// Value is the value itself.
	Value string
}

// NewFloat64Observation produces a new float-valued observation of the given kind.
func NewFloat64Observation(kind string, value float64) *Observation {
	return &Observation{
		MetricKind: kind,
		ValueType:  ValueTypeNumber,
		Value:      fmt.Sprintf("%f", value),
	}
}

// NewInt64Observation produces a new int-valued observation of the given kind.
func NewInt64Observation(kind string, value int64) *Observation {
	return &Observation{
		MetricKind: kind,
		ValueType:  ValueTypeNumber,
		Value:      fmt.Sprintf("%d", value),
	}
}

// NewStringObservation produces a new string-valued observation of the given kind.
func NewStringObservation(kind string, value string) *Observation {
	return &Observation{
		MetricKind: kind,
		ValueType:  ValueTypeString,
		Value:      value,
	}
}

// A Query is a collection of time-bounded search criteria for actions on DUTs.
type Query struct {
	// StartTime is the starting time for the query as a unix timestamp.
	StartTime time.Time
	// StopTime is the ending time for the query as a unix timestamp.
	StopTime time.Time
	// AssetTag is the asset tag for the DUT in question.
	AssetTag string
	// Hostname is the hostname for the DUT in question.
	// The hostname is less reliable than the asset tag because
	// it identifies a location rather than a device per se.
	Hostname string
	// Kind filters the actions by the "ActionKind" field.
	ActionKind string
	// Limit imposes a limit on the total number of actions returned.
	Limit int
	// PageToken is an opaque blob of data that is used to start the query at a specific point.
	PageToken string
	// OrderDescending controls how the result set should be ordered by time
	OrderDescending bool
}

// Lower takes a query and lowers it to a string using the filter syntax that Karte accepts.
// See karte/api/filter_syntax.md for more information.
func (q *Query) Lower() (string, error) {
	if q == nil {
		return "", nil
	}
	var out []string
	// Keep this list of if-statements up-to-date with the
	if !q.StartTime.IsZero() {
		return "", errors.Reason("lower: not yet implemented").Err()
	}
	if !q.StopTime.IsZero() {
		return "", errors.Reason("lower: not yet implemented").Err()
	}
	if q.AssetTag != "" {
		return "", errors.Reason("lower: not yet implemented").Err()
	}
	if q.Hostname != "" {
		out = append(out, fmt.Sprintf(`hostname == %q`, q.Hostname))
	}
	if q.ActionKind != "" {
		out = append(out, fmt.Sprintf(`kind == %q`, q.ActionKind))
	}
	// q.Limit is intentionally ignored for the purposes of generating a query.
	if q.PageToken != "" {
		return "", errors.Reason("lower: not yet implemented").Err()
	}
	filter := strings.Join(out, " && ")
	return filter, nil
}

// NewLastActionQuery returns a query for the last record of a given kind for the asset in question.
func NewLastActionQuery(assetTag string, kind string) *Query {
	return &Query{
		AssetTag:   assetTag,
		ActionKind: kind,
		Limit:      1,
	}
}

// NewLastActionBeforeTimeQuery returns a query for the last record before the stop time of a given kind
// for the asset in question.
func NewLastActionBeforeTimeQuery(assetTag string, kind string, stopTime time.Time) *Query {
	return &Query{
		AssetTag:   assetTag,
		ActionKind: kind,
		Limit:      1,
		StopTime:   stopTime,
	}
}

// NewListActionsInRangeQuery lists the actions for a given asset and given range in order.
//
// Sample usage:
//
//   q := NewListActionsInRangeQuery(..., "token1", 10)
//   res, err := metrics.Search(ctx, q)
//   if err != nil {
//      ...
//   }
//   q = NewListActionsInRangeQuery(..., res.PageToken, 10)
//   res, err = metrics.Search(ctx, q)
//   ...
//
func NewListActionsInRangeQuery(assetTag string, kind string, startTime time.Time, stopTime time.Time, pageToken string, limit int) *Query {
	return &Query{
		AssetTag:   assetTag,
		ActionKind: kind,
		StartTime:  startTime,
		StopTime:   stopTime,
		PageToken:  pageToken,
	}
}

// A QueryResult is the result of running a query.
type QueryResult struct {
	// Actions are the actions satisfying the criteria in question.
	Actions []*Action
	// PageToken is the token for resuming the query, if such a token exists.
	PageToken string
}

// Metrics is a simple interface for logging
// structured events and metrics.
type Metrics interface {
	// Create takes an action and creates it on the Karte side.
	// On success, it updates its action argument to reflect the Karte state.
	// Local versions of Create should emulate this.
	Create(ctx context.Context, action *Action) error

	// Update an action with observations.
	Update(ctx context.Context, action *Action) error

	// Search lists all the actions matching a set of constraints, up to
	// a limit on the number of returned actions.
	Search(ctx context.Context, q *Query) (*QueryResult, error)
}
