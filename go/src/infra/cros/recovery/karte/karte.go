// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package karte

import (
	"context"

	"go.chromium.org/luci/common/errors"

	kartepb "infra/cros/karte/api"
	kclient "infra/cros/karte/client"
	"infra/cros/recovery/logger/metrics"
)

// Client is a wrapped Karte client that exposes only the metrics.Metrics interface.
type client struct {
	impl kartepb.KarteClient
}

// NewMetrics creates a new metrics client.
func NewMetrics(ctx context.Context, c *kclient.Config, o ...kclient.Option) (metrics.Metrics, error) {
	innerClient, err := kclient.NewClient(ctx, c, o...)
	if err != nil {
		return nil, errors.Annotate(err, "wrap karte client").Err()
	}
	return &client{impl: innerClient}, nil
}

// Create creates an action and returns the action that was just created.
// Note that an action contains zero or more observations in it and that observations are not
// separate.
func (c *client) Create(ctx context.Context, action *metrics.Action) (*metrics.Action, error) {
	karteResp, err := c.impl.CreateAction(
		ctx,
		&kartepb.CreateActionRequest{
			Action: convertActionToKarteAction(action),
		},
	)
	if err != nil {
		return nil, errors.Annotate(err, "karte create").Err()
	}
	return convertKarteActionToAction(karteResp), nil
}

// Update takes an action and updates the entry in the Karte service, the source of truth.
// TODO(gregorynisbet): This implementation is not complete. A metrics action has observations attached to it.
// Updating Karte will require inspecting those observations and potentially updating or replacing them.
func (c *client) Update(ctx context.Context, action *metrics.Action) (*metrics.Action, error) {
	a := convertActionToKarteAction(action)
	karteResp, err := c.impl.UpdateAction(
		ctx,
		&kartepb.UpdateActionRequest{
			Action:     a,
			UpdateMask: nil,
		},
	)
	if err != nil {
		return nil, errors.Annotate(err, "karte update").Err()
	}
	return convertKarteActionToAction(karteResp), nil
}

// defaultResultSetSize is the number of records to return by default from Karte.
const defaultResultSetSize = 1000

// Search takes a query struct and produces a resultset.
func (c *client) Search(ctx context.Context, q *metrics.Query) (*metrics.QueryResult, error) {
	filter, lErr := q.Lower()
	if lErr != nil {
		return nil, errors.Annotate(lErr, "karte search").Err()
	}
	pageSize := q.Limit
	if pageSize <= 0 {
		pageSize = defaultResultSetSize
	}
	karteResp, kErr := c.impl.ListActions(
		ctx,
		&kartepb.ListActionsRequest{
			PageSize: defaultResultSetSize,
			// We explicitly do not set a page token so that we get
			// the most recent results first.
			PageToken: "",
			Filter:    filter,
		},
	)
	if kErr != nil {
		return nil, errors.Annotate(kErr, "karte search").Err()
	}

	var actions []*metrics.Action
	for _, a := range karteResp.GetActions() {
		action := convertKarteActionToAction(a)
		if action != nil {
			actions = append(actions, action)
		}
	}

	return &metrics.QueryResult{
		PageToken: karteResp.GetNextPageToken(),
		Actions:   actions,
	}, nil
}
