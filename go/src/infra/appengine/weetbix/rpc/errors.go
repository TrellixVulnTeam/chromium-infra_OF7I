// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"go.chromium.org/luci/grpc/appstatus"
	"google.golang.org/grpc/codes"
)

// invalidArgumentError annotates err as having an invalid argument.
// The error message is shared with the requester as is.
//
// Note that this differs from FailedPrecondition. It indicates arguments
// that are problematic regardless of the state of the system
// (e.g., a malformed file name).
func invalidArgumentError(err error) error {
	return appstatus.Attachf(err, codes.InvalidArgument, "%s", err)
}

// failedPreconditionError annotates err as failing a predondition for the
// operation. The error message is shared with the requester as is.
//
// See codes.FailedPrecondition for more context about when this
// should be used compared to invalid argument.
func failedPreconditionError(err error) error {
	return appstatus.Attachf(err, codes.FailedPrecondition, "%s", err)
}
