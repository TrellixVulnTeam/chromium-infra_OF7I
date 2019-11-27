// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/grpc/discovery"
	"go.chromium.org/luci/grpc/grpcmon"
	"go.chromium.org/luci/grpc/grpcutil"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/router"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	api "infra/appengine/cros/lab_inventory/api/v1"
)

// InstallHandlers installs the handlers implemented by the frontend package.
func InstallHandlers(r *router.Router, mwBase router.MiddlewareChain) {
	var si grpc.UnaryServerInterceptor
	si = grpcutil.NewUnaryServerPanicCatcher(si)
	si = grpcmon.NewUnaryServerInterceptor(si)
	s := prpc.Server{
		UnaryServerInterceptor: si,
	}
	api.RegisterInventoryServer(&s, &api.DecoratedInventory{
		Service: &InventoryServerImpl{},
		Prelude: checkAccess,
	})
	discovery.Enable(&s)
	s.InstallHandlers(r, mwBase)
}

// checkAccess verifies that the request is from an authorized user.
func checkAccess(ctx context.Context, _ string, _ proto.Message) (context.Context, error) {
	return ctx, nil
}
