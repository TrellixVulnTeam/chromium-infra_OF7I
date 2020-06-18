// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"

	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/frontend"
)

// SupportedClientMajorVersionNumber indicates the minimum client version
// supported by this server
//
// any client with major version number lower than this number will get an
// error to update their client to this major version or above.
const SupportedClientMajorVersionNumber = 1

func main() {
	modules := []module.Module{
		gaeemulation.NewModuleFromFlags(),
	}

	cfgLoader := config.Loader{}
	cfgLoader.RegisterFlags(flag.CommandLine)

	server.Main(nil, modules, func(srv *server.Server) error {
		// Load service config form a local file (deployed via GKE),
		// periodically reread it to pick up changes without full restart.
		if _, err := cfgLoader.Load(); err != nil {
			return err
		}
		srv.RunInBackground("ufs.config", cfgLoader.ReloadLoop)

		srv.Context = config.Use(srv.Context, cfgLoader.Config())
		srv.RegisterUnaryServerInterceptor(versionInterceptor)
		frontend.InstallServices(srv.PRPC)
		return nil
	})
}

// versionInterceptor interceptor to handle client version check per RPC call
func versionInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "Retrieving metadata failed.")
	}
	version, ok := md["clientversion"]
	if !ok || len(version) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Client major version not supplied.")
	}
	major, err := strconv.ParseInt(version[0], 10, 32)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Client major version format error.")
	}
	if major < SupportedClientMajorVersionNumber {
		return nil, status.Errorf(codes.FailedPrecondition,
			fmt.Sprintf("Unsupported client version. Please update your client "+
				"version to v%d.X.X or above.", SupportedClientMajorVersionNumber))
	}

	// Calls the handler
	h, err := handler(ctx, req)

	return h, err
}
