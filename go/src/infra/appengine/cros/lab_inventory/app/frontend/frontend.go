// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/discovery"
	"go.chromium.org/luci/grpc/grpcmon"
	"go.chromium.org/luci/grpc/grpcutil"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/router"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/appengine/cros/lab_inventory/app/config"
)

// InstallHandlers installs the handlers implemented by the frontend package.
func InstallHandlers(r *router.Router, mwBase router.MiddlewareChain) {
	s := prpc.Server{
		UnaryServerInterceptor: grpcutil.ChainUnaryServerInterceptors(
			grpcmon.UnaryServerInterceptor,
			grpcutil.UnaryServerPanicCatcherInterceptor,
		),
	}
	api.RegisterInventoryServer(&s, &api.DecoratedInventory{
		Service: &InventoryServerImpl{},
		Prelude: checkAccess,
	})
	discovery.Enable(&s)
	s.InstallHandlers(r, mwBase)
}

// checkAccess verifies that the request is from an authorized user.
func checkAccess(ctx context.Context, rpcName string, _ proto.Message) (context.Context, error) {
	logging.Infof(ctx, "%s requests the RPC %s", auth.CurrentUser(ctx), rpcName)
	cfg := config.Get(ctx)
	var accessGroup *config.LuciAuthGroup
	switch rpcName {
	case "AddCrosDevices", "UpdateCrosDevicesSetup":
		accessGroup = cfg.GetSetupWriters()
	case "GetCrosDevices":
		accessGroup = cfg.GetReaders()
	case "UpdateDutsStatus":
		accessGroup = cfg.GetStatusWriters()
	case "DeleteCrosDevices":
		accessGroup = cfg.GetPrivilegedWriters()
	default:
		return ctx, status.Errorf(codes.Unimplemented, rpcName)
	}
	group := accessGroup.GetValue()
	allow, err := auth.IsMember(ctx, group)
	if err != nil {
		logging.Warningf(ctx, "Check group '%s' membership failed: %s", group, err.Error())
		return ctx, status.Errorf(codes.Internal, "can't check access group membership: %s", err)
	}
	if !allow {
		return ctx, status.Errorf(codes.PermissionDenied, "%s is not a member of %s", auth.CurrentIdentity(ctx), accessGroup)
	}
	return ctx, nil
}
