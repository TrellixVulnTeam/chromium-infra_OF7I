// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"regexp"
	"strconv"

	"go.chromium.org/luci/common/logging"
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
const SupportedClientMajorVersionNumber = 2

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
	userAgentExists, userAgentErr := validateUserAgent(md)
	if userAgentExists && userAgentErr != nil {
		return nil, userAgentErr
	}
	if !userAgentExists {
		return nil, status.Errorf(codes.InvalidArgument, "user-agent is not specified in the incoming request")
	}
	logging.Debugf(ctx, "Successfully pass user-agent version check: major version %d", SupportedClientMajorVersionNumber)
	return handler(ctx, req)
}

// Assuming the version number for major, minor and patch are less than 1000.
var versionRegex = regexp.MustCompile(`[0-9]{1,3}`)

// validateUserAgent returns a tuple
//     (if user-agent exists, if user-agent is valid)
func validateUserAgent(md metadata.MD) (bool, error) {
	version, ok := md["user-agent"]
	// Only check version for skylab commands which already set user-agent
	if ok {
		majors := versionRegex.FindAllString(version[0], 1)
		if len(majors) != 1 {
			return ok, status.Errorf(codes.InvalidArgument, "user-agent %s doesn't contain major version", version[0])
		}
		major, err := strconv.ParseInt(majors[0], 10, 32)
		if err != nil {
			return ok, status.Errorf(codes.InvalidArgument, "user-agent %s has wrong major version format", version[0])
		}
		if major < SupportedClientMajorVersionNumber {
			return ok, status.Errorf(codes.FailedPrecondition,
				fmt.Sprintf("Unsupported client version. Please update your client version to v%d.X.X or above.", SupportedClientMajorVersionNumber))
		}
	}
	return ok, nil
}
