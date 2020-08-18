// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"
	"strings"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	api "infra/unifiedfleet/api/v1/rpc"
)

// InstallServices installs ...
func InstallServices(apiServer *prpc.Server) {
	api.RegisterFleetServer(apiServer, &api.DecoratedFleet{
		Service: &FleetServerImpl{},
		Prelude: checkAccess,
	})
}

// checkAccess verifies that the request is from an authorized user.
func checkAccess(ctx context.Context, rpcName string, _ proto.Message) (context.Context, error) {
	logging.Debugf(ctx, "Check access for %s", rpcName)
	// TODO: Re-org the permissions with luci-realm
	group := []string{"mdb/chrome-fleet-software-team", "mdb/chrome-labs", "mdb/hwops-nsi"}
	switch rpcName {
	case "CreateMachineLSE", "UpdateMachineLSE", "CreateVM", "UpdateVM":
		group = []string{"mdb/chrome-labs", "mdb/chrome-fleet-software-team", "chromeos-inventory-setup-label-write-access", "machine-db-writers"}
	case "DeleteMachineLSE", "CreateVlan", "UpdateVlan", "DeleteVlan", "DeleteVM":
		group = []string{"mdb/chrome-labs", "mdb/chrome-fleet-software-team", "chromeos-inventory-privileged-access"}
	case "ListMachineLSEs", "GetMachineLSE", "ListVMs", "GetVM", "GetVlan", "ListVlans", "ListMachines":
		group = []string{"mdb/chrome-labs", "mdb/chrome-fleet-software-team", "machine-db-readers"}
	case "DeleteMachine":
		group = append(group, "mdb/hwops-nsi", "chromeos-inventory-privileged-access")
	case "GetMachine", "GetState":
		group = append(group, "chromeos-inventory-readonly-access", "machine-db-readers")
	case "UpdateState":
		group = append(group, "chromeos-inventory-status-label-write-access")
	}
	if strings.HasPrefix(rpcName, "Import") {
		group = []string{"mdb/chrome-fleet-software-team"}
	}
	allow, err := auth.IsMember(ctx, group...)
	if err != nil {
		logging.Errorf(ctx, "Check group '%s' membership failed: %s", group, err.Error())
		return ctx, status.Errorf(codes.Internal, "can't check access group membership: %s", err)
	}
	if !allow {
		return ctx, status.Errorf(codes.PermissionDenied, "%s is not a member of %s", auth.CurrentIdentity(ctx), group)
	}
	logging.Infof(ctx, "%s is a member of %s", auth.CurrentIdentity(ctx), group)
	return ctx, nil
}
