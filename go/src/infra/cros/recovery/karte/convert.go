// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package karte

import (
	"time"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	kartepb "infra/cros/karte/api"
	"infra/cros/recovery/logger/metrics"
)

// ConvertActionStatusToKarteActionStatus takes a metrics action status and converts it to a Karte action status.
func convertActionStatusToKarteActionStatus(status metrics.ActionStatus) kartepb.Action_Status {
	// TODO(gregorynisbet): Add support for skipped actions to Karte.
	switch status {
	case metrics.ActionStatusSuccess:
		return kartepb.Action_SUCCESS
	case metrics.ActionStatusFail:
		return kartepb.Action_FAIL
	default:
		return kartepb.Action_STATUS_UNSPECIFIED
	}
}

// ConvertKarteActionStatusToActionStatus takes a Karte action status and converts it to a metrics action status.
func convertKarteActionStatusToActionStatus(status kartepb.Action_Status) metrics.ActionStatus {
	switch status {
	case kartepb.Action_SUCCESS:
		return metrics.ActionStatusSuccess
	case kartepb.Action_FAIL:
		return metrics.ActionStatusFail
	default:
		return metrics.ActionStatusUnspecified
	}
}

// ConvertTimeToProtobufTimestamp takes a time and converts it to a pointer to a protobuf timestamp.
// This method sends the zero time to a nil pointer.
func convertTimeToProtobufTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

// ConvertProtobufTimestampToTime takes a protobuf timestamp and converts it
// to a Go time.Time. We can't just use thing.AsTime() because the protobuf timestamp and time.Time do not agree on what the "zero time" is.
//
// This is the error message that we get if we instead use thing.AsTime().
//
//   -   StartTime:      s"0001-01-01 00:00:00 +0000 UTC",
//   +   StartTime:      s"1970-01-01 00:00:00 +0000 UTC",
//   -   StopTime:       s"0001-01-01 00:00:00 +0000 UTC",
//   +   StopTime:       s"1970-01-01 00:00:00 +0000 UTC",
//
func convertProtobufTimestampToTime(t *timestamppb.Timestamp) time.Time {
	var zero time.Time
	if t == nil {
		return zero
	}
	return t.AsTime()
}

// ConvertActionToKarteAction takes an action and converts it to a Karte action.
func convertActionToKarteAction(action *metrics.Action) *kartepb.Action {
	if action == nil {
		return nil
	}
	return &kartepb.Action{
		Name:           action.Name,
		Kind:           action.ActionKind,
		SwarmingTaskId: action.SwarmingTaskID,
		BuildbucketId:  action.BuildbucketID,
		AssetTag:       action.AssetTag,
		StartTime:      convertTimeToProtobufTimestamp(action.StartTime),
		StopTime:       convertTimeToProtobufTimestamp(action.StopTime),
		Status:         convertActionStatusToKarteActionStatus(action.Status),
		FailReason:     action.FailReason,
		Hostname:       action.Hostname,
	}
}

// ConvertKarteActionToAction takes a Karte action and converts it to an action.
func convertKarteActionToAction(action *kartepb.Action) *metrics.Action {
	if action == nil {
		return nil
	}
	return &metrics.Action{
		Name:           action.GetName(),
		ActionKind:     action.GetKind(),
		SwarmingTaskID: action.GetSwarmingTaskId(),
		BuildbucketID:  action.GetBuildbucketId(),
		AssetTag:       action.GetAssetTag(),
		StartTime:      convertProtobufTimestampToTime(action.GetStartTime()),
		StopTime:       convertProtobufTimestampToTime(action.GetStopTime()),
		Status:         convertKarteActionStatusToActionStatus(action.GetStatus()),
		FailReason:     action.GetFailReason(),
		Hostname:       action.GetHostname(),
	}
}
