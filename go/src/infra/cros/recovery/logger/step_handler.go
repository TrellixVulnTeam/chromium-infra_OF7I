// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package logger

import (
	"context"
)

// StepHandler represents a simple interface for reporting steps.
type StepHandler interface {
	// StartStep starts new step with provided name.
	// The returned context is updated so that calling StartStep on it will create sub-steps.
	// https://pkg.go.dev/go.chromium.org/luci/luciexe/build#StartStep
	StartStep(ctx context.Context, name string) (Step, context.Context)
}

// Step represents a single step to track
type Step interface {
	Close(ctx context.Context, err error)
}
