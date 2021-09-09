// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// The metrics implementation inside the logger package is a default implementation
// of the Metrics interface. It will never talk to external services.
// It is intended only for local development.
package logger

import (
	"context"
	"encoding/json"

	"go.chromium.org/luci/common/errors"
)

// metrics is a default Metric implementation that logs all events
// to the logger.
type metrics struct {
	logger Logger
}

// NewLogMetrics creates a default metric sink.
func NewLogMetrics(l Logger) Metrics {
	return &metrics{
		logger: l,
	}
}

// Create marshals an action as JSON and logs it at the debug level.
func (m *metrics) Create(ctx context.Context, action *Action) (*Action, error) {
	a, err := json.MarshalIndent(action, "", "    ")
	if err != nil {
		// TODO(gregorynisbet): Check if action is nil.
		return nil, errors.Annotate(err, "record action for asset %q", action.AssetTag).Err()
	}
	m.logger.Debug("Create action %q: %s\n", action.ActionKind, string(a))
	return action, nil
}

// Search lists the actions matching a given criterion.
func (m *metrics) Search(ctx context.Context, q *Query) (*QueryResult, error) {
	return nil, errors.New("list actions matching: not yet implemented")
}
