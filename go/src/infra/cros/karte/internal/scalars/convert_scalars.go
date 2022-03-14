// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package scalars

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	kartepb "infra/cros/karte/api"
)

// ConvertTimestampPtrToTime takes a pointer to a timestamp proto value and
// converts it to a Go time.Time value, sending nil to zero.
func ConvertTimestampPtrToTime(timestamp *timestamppb.Timestamp) time.Time {
	var out time.Time
	if timestamp != nil {
		out = timestamp.AsTime()
	}
	return out
}

// ConvertTimeToTimestampPtr takes a time value and converts it to a timestamp
// proto value. It sends the zero value for time.Time to a nil pointer.
func ConvertTimeToTimestampPtr(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func ConvertTimestampPtrToString(timestamp *timestamppb.Timestamp) string {
	if timestamp == nil {
		return ""
	}
	return fmt.Sprintf("%d:%d", timestamp.GetSeconds(), timestamp.GetNanos())
}

// ConvertActionStatusToInt32 takes an action status and converts it to an int32.
// If this function is passed an invalid kartepb.Action_status, the results are undefined.
func ConvertActionStatusToInt32(status kartepb.Action_Status) int32 {
	return int32(status)
}

// ConvertActionStatusToInt32 takes an int32 and converts it to an action status.
// If this function is passed a integer that is out of range, the results are undefined.
func ConvertInt32ToActionStatus(i int32) kartepb.Action_Status {
	return kartepb.Action_Status(i)
}

// ConvertActionStatusIntToString takes an int32 and converts it to the string representation of an action status.
// If this function is passed a integer that is out of range, the results are undefined.
func ConvertActionStatusIntToString(i int32) string {
	return kartepb.Action_Status_name[i]
}
