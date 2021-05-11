// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"

	"go.chromium.org/luci/grpc/prpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"

	kartepb "infra/cros/karte/api"
)

// karteFrontend is the implementation of kartepb.KarteServer
// used in the application.
type karteFrontend struct{}

// NewKarteFrontend produces a new Karte frontend.
func NewKarteFrontend() kartepb.KarteServer {
	return &karteFrontend{}
}

// ListActions lists the actions that Karte knows about.
func (k *karteFrontend) ListActions(ctx context.Context, req *kartepb.ListActionsRequest) (*kartepb.ListActionsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "ListActions not implemented")
}

// ListObservations lists the observations that Karte knows about.
func (k *karteFrontend) ListObservations(context.Context, *kartepb.ListObservationsRequest) (*kartepb.ListObservationsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "ListObservations not implemented")
}

// InstallServices takes a Karte frontend and exposes it to a LUCI prpc.Server.
func InstallServices(srv *prpc.Server) {
	kartepb.RegisterKarteServer(srv, NewKarteFrontend())
}
