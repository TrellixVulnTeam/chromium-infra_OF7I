// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsNotFoundError checks if an error has code NOT_FOUND
func IsNotFoundError(err error) bool {
	s, ok := status.FromError(err)
	if ok && s.Code() == codes.NotFound {
		return true
	}
	return false
}

// IsInternalError checks if an error has code INTERNAL
func IsInternalError(err error) bool {
	s, ok := status.FromError(err)
	if ok && s.Code() == codes.Internal {
		return true
	}
	return false
}
