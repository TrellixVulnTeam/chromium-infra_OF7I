// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package types contains types common to execution sub-packages.
package types

import (
	"fmt"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
)

// InvocationID is a unique identifier used to refer to the input invocations.
//
// InvocationID is required because there is no natural unique name for
// invocations.
type InvocationID string

// NewInvocationID constructs a new unique ID for an invocation.
//
// The first argument should be used to ensure that multiple invocations for the
// same test get a distinct InvocationID.
func NewInvocationID(i int, test *steps.EnumerationResponse_AutotestInvocation) InvocationID {
	return InvocationID(fmt.Sprintf("%d_%s", i, test.GetTest().GetName()))
}

// TaskDimKeyVal is a key-value Swarming task dimension.
type TaskDimKeyVal struct {
	Key, Val string
}
