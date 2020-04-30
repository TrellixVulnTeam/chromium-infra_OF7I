// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package api

import (
	"fmt"

	"infra/cros/cmd/lucifer/internal/logdog"
)

// Logger returns the root LogDog logger.
func (c *Client) Logger() logdog.Logger {
	return wrappedLogger{
		Logger: c.logger,
		client: c,
	}
}

// Step creates and returns a LogDog step under the currently active
// step, or the root logger if no step is active.
func (c *Client) Step(name string) logdog.Step {
	if c.step == nil {
		return c.Logger().Step(name)
	}
	return c.step.Step(name)
}

// wrappedLogger wraps a LogDog logger and overrides Step so that the
// current Step is tracked in Client.
type wrappedLogger struct {
	logdog.Logger
	client *Client
}

func (lg wrappedLogger) Step(name string) logdog.Step {
	s := wrappedStep{
		logdogStep: lg.Logger.Step(name),
		client:     lg.client,
	}
	lg.client.step = s
	return s
}

// logdogStep is a type alias for embedding in wrappedStep to avoid a
// name conflict with the Step method.
type logdogStep = logdog.Step

// wrappedStep wraps a LogDog step and overrides Step so that the
// current Step is tracked in Client.
type wrappedStep struct {
	logdogStep
	client *Client
	parent logdog.Step
}

func (s wrappedStep) Step(name string) logdog.Step {
	s2 := wrappedStep{
		logdogStep: s.logdogStep.Step(name),
		client:     s.client,
		parent:     s,
	}
	s.client.step = s2
	return s2
}

func (s wrappedStep) Close() error {
	if err := s.logdogStep.Close(); err != nil {
		panic(fmt.Sprintf("Step.Close returned error (should always be nil): %s", err))
	}
	s.client.step = s.parent
	return nil
}
