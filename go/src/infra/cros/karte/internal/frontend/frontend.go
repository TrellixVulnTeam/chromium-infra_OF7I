// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	kartepb "infra/cros/karte/api"
)

// karteFrontend is the implementation of kartepb.KarteServer
// used in the application.
type karteFrontend struct{}

// NewKarteFrontend produces a new Karte frontend.
func NewKarteFrontend() kartepb.KarteServer {
	return &karteFrontend{}
}

// CreateAction creates an action, stores it in datastore, and then returns the just-created action.
func (k *karteFrontend) CreateAction(ctx context.Context, req *kartepb.CreateActionRequest) (*kartepb.Action, error) {
	if req == nil {
		return nil, errors.New("create action: request is nil")
	}
	if req.GetAction() == nil {
		return nil, errors.New("create action: action is nil")
	}
	actionEntity, err := ConvertActionToActionEntity(req.GetAction())
	if err != nil {
		return nil, errors.Annotate(err, "create action").Err()
	}
	if err := PutActionEntities(ctx, actionEntity); err != nil {
		return nil, errors.Annotate(err, "writing action to datastore").Err()
	}
	return req.GetAction(), nil
}

// CreateObservation creates an observation and then returns the just-created observation.
func (k *karteFrontend) CreateObservation(ctx context.Context, req *kartepb.CreateObservationRequest) (*kartepb.Observation, error) {
	if req == nil {
		return nil, errors.New("create observation: request is nil")
	}
	if req.GetObservation() == nil {
		return nil, errors.New("create observation: observation is nil")
	}
	observationEntity, err := ConvertObservationToObservationEntity(req.GetObservation())
	if err != nil {
		return nil, errors.Annotate(err, "create observation").Err()
	}
	if err := PutObservationEntities(ctx, observationEntity); err != nil {
		return nil, errors.Annotate(err, "writing action to datastore").Err()
	}
	return req.GetObservation(), nil
}

// ListActions lists the actions that Karte knows about.
func (k *karteFrontend) ListActions(ctx context.Context, req *kartepb.ListActionsRequest) (*kartepb.ListActionsResponse, error) {
	q := MakeAllActionEntitiesQuery(req.GetPageToken())
	es, err := q.Next(ctx, req.GetPageSize())
	if err != nil {
		return nil, errors.Annotate(err, "list actions (page size: %d)", req.GetPageSize()).Err()
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
		return nil, errors.Annotate(err, "list observations (page size: %d)", req.GetPageSize()).Err()
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
