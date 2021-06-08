// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"

	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	kartepb "infra/cros/karte/api"
)

// karteFrontend is the implementation of kartepb.KarteServer
// used in the application.
type karteFrontend struct{}

// NewKarteFrontend produces a new Karte frontend.
func NewKarteFrontend() kartepb.KarteServer {
	return &karteFrontend{}
}

// CreateAction creates an action and then returns the just-created action.
// TODO(gregorynisbet): Replace CreateAction with a real implementation.
func (k *karteFrontend) CreateAction(ctx context.Context, req *kartepb.CreateActionRequest) (*kartepb.Action, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// ListActions lists the actions that Karte knows about.
func (k *karteFrontend) ListActions(ctx context.Context, req *kartepb.ListActionsRequest) (*kartepb.ListActionsResponse, error) {
	q := MakeAllActionEntitiesQuery(req.GetPageToken())
	es, err := q.Next(ctx, req.GetPageSize())
	if err != nil {
		return nil, err
	}
	var actions []*kartepb.Action
	for _, e := range es {
		actions = append(actions, e.ConvertToAction())
	}
	return &kartepb.ListActionsResponse{
		Actions:       actions,
		NextPageToken: q.Token,
	}, nil
}

// ListObservations lists the observations that Karte knows about.
func (k *karteFrontend) ListObservations(ctx context.Context, req *kartepb.ListObservationsRequest) (*kartepb.ListObservationsResponse, error) {
	q := MakeAllObservationEntitiesQuery(req.GetPageToken())
	es, err := q.Next(ctx, req.GetPageSize())
	if err != nil {
		return nil, err
	}
	var observations []*kartepb.Observation
	for _, e := range es {
		observations = append(observations, e.ConvertToObservation())
	}
	return &kartepb.ListObservationsResponse{
		Observations:  observations,
		NextPageToken: q.Token,
	}, nil
}

// InstallServices takes a Karte frontend and exposes it to a LUCI prpc.Server.
func InstallServices(srv *prpc.Server) {
	kartepb.RegisterKarteServer(srv, NewKarteFrontend())
}
