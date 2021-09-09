// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package karte

import (
	"context"
	kartepb "infra/cros/karte/api"

	"go.chromium.org/luci/common/errors"

	kclient "infra/cros/karte/client"
	"infra/cros/recovery/logger"
)

// Client is a wrapped Karte client that exposes only the logger.Metrics interface.
type client struct {
	impl kartepb.KarteClient
}

// NewMetrics creates a new metrics client.
func NewMetrics(ctx context.Context, c *kclient.Config, o ...kclient.Option) (logger.Metrics, error) {
	innerClient, err := kclient.NewClient(ctx, c, o...)
	if err != nil {
		return nil, errors.Annotate(err, "wrap karte client").Err()
	}
	return &client{impl: innerClient}, nil
}

// Create creates an action and returns the action that was just created.
// Note that an action contains zero or more observations in it and that observations are not
// separate.
func (c *client) Create(ctx context.Context, action *logger.Action) (*logger.Action, error) {
	panic("not implemented")
}

// Search takes a query struct and produces a resultset.
func (c *client) Search(ctx context.Context, q *logger.Query) (*logger.QueryResult, error) {
	panic("not implemented")
}
