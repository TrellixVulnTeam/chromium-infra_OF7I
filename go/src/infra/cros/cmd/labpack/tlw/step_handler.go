// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tlw

import (
	"context"

	"go.chromium.org/luci/luciexe/build"

	"infra/cros/recovery/logger"
)

// NewStepper create new instance of StepHandler for recovery engine.
func NewStepHandler(logger logger.Logger) logger.StepHandler {
	return &stepHandler{
		logger: logger,
	}
}

// stepHandler is implementation of StepHandler insterface for recovery engine.
type stepHandler struct {
	logger logger.Logger
}

// StartStep starts new step with provided name.
// The returned context is updated so that calling StartStep on it will create sub-steps.
// https://pkg.go.dev/go.chromium.org/luci/luciexe/build#StartStep
func (s *stepHandler) StartStep(ctx context.Context, name string) (logger.Step, context.Context) {
	step, ctx := build.StartStep(ctx, "Input params")
	s.logger.Debug("Step %q: started.", name)
	return &simpleStep{
		logger: s.logger,
		step:   step,
		closed: false,
		name:   name,
	}, ctx
}

// simpleStep is implementation of Step insterface for recovery engine.
type simpleStep struct {
	name   string
	closed bool
	step   *build.Step
	logger logger.Logger
}

// Close close the step and the step cannot be reused after that.
func (s *simpleStep) Close(ctx context.Context, err error) {
	if s.closed {
		s.logger.Warning("Step %q: the step already closed.", s.name)
		return
	}
	s.step.End(err)
	s.closed = true
	s.logger.Debug("Step %q: closed.", s.name)
}
