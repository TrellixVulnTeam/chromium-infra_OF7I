// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
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

// CreateAction creates an action, stores it in datastore, and then returns the just-created action.
func (k *karteFrontend) CreateAction(ctx context.Context, req *kartepb.CreateActionRequest) (*kartepb.Action, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "create action: request is nil")
	}
	if req.GetAction() == nil {
		return nil, status.Errorf(codes.InvalidArgument, "create action: action is nil")
	}
	logging.Infof(ctx, "Creating action of kind %q", req.GetAction().GetKind())
	actionEntity, err := ConvertActionToActionEntity(req.GetAction())
	if err != nil {
		logging.Errorf(ctx, "error converting action: %s", err)
		return nil, errors.Annotate(err, "create action").Err()
	}
	if err := PutActionEntities(ctx, actionEntity); err != nil {
		logging.Errorf(ctx, "error writing action: %s", err)
		return nil, errors.Annotate(err, "writing action to datastore").Err()
	}
	return req.GetAction(), nil
}

// CreateObservation creates an observation and then returns the just-created observation.
func (k *karteFrontend) CreateObservation(ctx context.Context, req *kartepb.CreateObservationRequest) (*kartepb.Observation, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "create observation: request is nil")
	}
	if req.GetObservation() == nil {
		return nil, status.Errorf(codes.InvalidArgument, "create observation: observation is nil")
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
	q, err := newActionEntitiesQuery(req.GetPageToken(), req.GetFilter())
	if err != nil {
		return nil, errors.Annotate(err, "list actions").Err()
	}

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
	q, err := newObservationEntitiesQuery(req.GetPageToken(), req.GetFilter())
	if err != nil {
		return nil, errors.Annotate(err, "list observations").Err()
	}
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

// UpdateAction updates an action in datastore and creates it if necessary when allow_missing is set.
func (k *karteFrontend) UpdateAction(ctx context.Context, req *kartepb.UpdateActionRequest) (*kartepb.Action, error) {
	reqActionEntity, err := ConvertActionToActionEntity(req.GetAction())
	if err != nil {
		return nil, errors.Annotate(err, "update action").Err()
	}
	entity, err := UpdateActionEntity(
		ctx,
		reqActionEntity,
		req.GetUpdateMask().GetPaths(),
	)
	return entity.ConvertToAction(), err
}

// InstallServices takes a Karte frontend and exposes it to a LUCI prpc.Server.
func InstallServices(srv *prpc.Server) {
	kartepb.RegisterKarteServer(srv, NewKarteFrontend())
}
