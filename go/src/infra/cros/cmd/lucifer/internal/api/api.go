// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package api contains the interface that Lucifer uses to talk to the
// outside world.
//
// This includes LogDog-aware logging (e.g., support for steps even if
// LogDog is unavailable) and metrics.
package api

import (
	"infra/cros/cmd/lucifer/internal/logdog"
)

// Client provides the interface that Lucifer uses to talk to the
// outside world.
//
// Client tracks the current/last LogDog step that was created.  If
// LogDog steps are created or closed in an unusual order, the
// behavior of this current step tracking is undefined.  Note that
// LogDog is synchronous and cannot handle this anyway.  See the
// logdog package for details on this behavior.
type Client struct {
	logger logdog.Logger
	step   logdog.Step
}

// NewClient returns a new client.
func NewClient(lg logdog.Logger) *Client {
	return &Client{
		logger: lg,
	}
}
