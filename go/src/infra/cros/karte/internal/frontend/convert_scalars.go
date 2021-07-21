// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	kartepb "infra/cros/karte/api"
)

// ConvertTimestampPtrToTime takes a pointer to a timestamp proto value and
// converts it to a Go time.Time value, sending nil to zero.
func convertTimestampPtrToTime(timestamp *timestamppb.Timestamp) time.Time {
	var out time.Time
	if timestamp != nil {
		out = timestamp.AsTime()
	}
	return out
}

// ConvertTimeToTimestampPtr takes a time value and converts it to a timestamp
// proto value. It sends the zero value for time.Time to a nil pointer.
func convertTimeToTimestampPtr(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

// ConvertActionStatusToInt32 takes an action status and converts it to an int32.
func convertActionStatusToInt32(status kartepb.Action_Status) int32 {
	// TODO(gregorynisbet): Add validation.
	return int32(status)
}

// ConvertActionStatusToInt32 takes an int32 and converts it to an action status.
func convertInt32ToActionStatus(i int32) kartepb.Action_Status {
	// TODO(gregorynisbet): Add validation.
	return kartepb.Action_Status(i)
}
