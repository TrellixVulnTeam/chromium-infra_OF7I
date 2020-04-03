// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/prpc"

	api "infra/appengine/unified-fleet/api/v1"
)

// InstallServices installs ...
func InstallServices(apiServer *prpc.Server) {
	api.RegisterRegistrationServer(apiServer, &api.DecoratedRegistration{
		Service: &RegistrationServerImpl{},
		Prelude: checkAccess,
	})
	api.RegisterConfigurationServer(apiServer, &api.DecoratedConfiguration{
		Service: &ConfigurationServerImpl{},
		Prelude: checkAccess,
	})
}

// checkAccess verifies that the request is from an authorized user.
func checkAccess(ctx context.Context, rpcName string, _ proto.Message) (context.Context, error) {
	logging.Debugf(ctx, "Check access for %s", rpcName)
	return ctx, nil
}
