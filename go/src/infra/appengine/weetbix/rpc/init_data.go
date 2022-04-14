// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"context"

	"go.chromium.org/luci/server/auth"

	"infra/appengine/weetbix/internal/config"
	pb "infra/appengine/weetbix/proto/v1"
)

// A server that provides the data to initialize the client.
type initDataGeneratorServer struct{}

// Creates a new initialization data server.
func NewInitDataGeneratorServer() *pb.DecoratedInitDataGenerator {
	return &pb.DecoratedInitDataGenerator{
		Prelude:  checkAllowedPrelude,
		Service:  &initDataGeneratorServer{},
		Postlude: gRPCifyAndLogPostlude,
	}
}

// Gets the initialization data.
func (*initDataGeneratorServer) GenerateInitData(ctx context.Context, request *pb.GenerateInitDataRequest) (*pb.GenerateInitDataResponse, error) {
	logoutURL, err := auth.LogoutURL(ctx, request.ReferrerUrl)
	if err != nil {
		return nil, err
	}

	config, err := config.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.GenerateInitDataResponse{
		InitData: &pb.InitData{
			Hostnames: &pb.Hostnames{
				MonorailHostname: config.MonorailHostname,
			},
			User: &pb.User{
				Email: auth.CurrentUser(ctx).Email,
			},
			AuthUrls: &pb.AuthUrls{
				LogoutUrl: logoutURL,
			},
		},
	}, nil
}
