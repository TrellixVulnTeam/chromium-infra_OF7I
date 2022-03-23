// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// The metrics implementation inside the package of the same name is a default implementation
// of the Metrics interface. It will never talk to external services.
// It is intended only for local development.
package metrics

import (
	"context"
	"encoding/json"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/logger"
)

// metrics is a default Metric implementation that logs all events
// to the logger.
type metrics struct {
	logger logger.Logger
}

// NewLogMetrics creates a default metric sink.
func NewLogMetrics(l logger.Logger) Metrics {
	return &metrics{
		logger: l,
	}
}

// Create marshals an action as JSON and logs it at the debug level.
// If the method receiver is nil, the behavior is undefined.
func (m *metrics) Create(ctx context.Context, action *Action) error {
	if m == nil {
		return errors.Reason("metrics create: metrics cannot be nil").Err()
	}
	a, err := json.MarshalIndent(action, "", "    ")
	if err != nil {
		// TODO(gregorynisbet): Check if action is nil.
		return errors.Annotate(err, "record action for asset %q", action.AssetTag).Err()
	}
	m.logger.Debugf("Create action %q: %s\n", action.ActionKind, string(a))
	return nil
}

// Update marshals an action as JSON and logs it at the debug level.
// TODO(gregorynisbet): Consider replacing the default implementation with an in-memory implementation of Karte.
// If the method receiver is nil, the behavior is undefined.
func (m *metrics) Update(ctx context.Context, action *Action) error {
	if m == nil {
		return errors.Reason("metrics update: metrics cannot be nil").Err()
	}
	if action == nil {
		return errors.Reason("metrics update: action cannot be nil").Err()
	}
	a, err := json.MarshalIndent(action, "", "    ")
	if err != nil {
		// TODO(gregorynisbet): Check if action is nil.
		return errors.Annotate(err, "record action for asset %q", action.AssetTag).Err()
	}
	m.logger.Debugf("Update action %q: %s\n", action.ActionKind, string(a))
	return nil
}

// Search lists the actions matching a given criterion.
// If the method receiver is nil, the behavior is undefined.
func (m *metrics) Search(ctx context.Context, q *Query) (*QueryResult, error) {
	if m == nil {
		return nil, errors.Reason("metrics search: metrics cannot be nil").Err()
	}
	return nil, errors.New("list actions matching: not yet implemented")
}
