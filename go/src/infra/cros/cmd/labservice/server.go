// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/cros/cmd/labservice/internal/log"
)

type server struct {
	labapi.UnimplementedInventoryServiceServer
}

func (s *server) GetDutTopology(req *labapi.GetDutTopologyRequest, stream labapi.InventoryService_GetDutTopologyServer) error {
	ctx := stream.Context()
	log.Infof(ctx, "GetDutTopology %v", req.GetId())
	return status.Error(codes.Unimplemented, "unimplemented")
}
