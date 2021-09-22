// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package karte

import (
	kartepb "infra/cros/karte/api"
	"infra/cros/recovery/logger"
	"time"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// ConvertActionStatusToKarteActionStatus takes a logger action status and converts it to a Karte action status.
func convertActionStatusToKarteActionStatus(status logger.ActionStatus) kartepb.Action_Status {
	// TODO(gregorynisbet): Add support for skipped actions to Karte.
	switch status {
	case logger.ActionStatusSuccess:
		return kartepb.Action_SUCCESS
	case logger.ActionStatusFail:
		return kartepb.Action_FAIL
	default:
		return kartepb.Action_STATUS_UNSPECIFIED
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

// ConvertActionToKarteAction takes an action and converts it to a Karte action.
func convertActionToKarteAction(action *logger.Action) *kartepb.Action {
	if action == nil {
		return nil
	}
	return &kartepb.Action{
		Name:           action.Name,
		Kind:           action.ActionKind,
		SwarmingTaskId: action.SwarmingTaskID,
		AssetTag:       action.AssetTag,
		StartTime:      convertTimeToProtobufTimestamp(action.StartTime),
		StopTime:       convertTimeToProtobufTimestamp(action.StopTime),
		Status:         convertActionStatusToKarteActionStatus(action.Status),
		FailReason:     action.FailReason,
	}
}
