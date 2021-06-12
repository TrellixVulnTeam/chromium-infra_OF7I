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
	"go.chromium.org/luci/server/router"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	api "infra/unifiedfleet/api/v1/rpc"
)

// InstallServices installs ...
func InstallServices(apiServer *prpc.Server) {
	apiServer.AccessControl = prpc.AllowOriginAll
	api.RegisterFleetServer(apiServer, &api.DecoratedFleet{
		Service: &FleetServerImpl{},
		Prelude: checkAccess,
	})
}

// InstallHandlers installs non PRPC handlers
func InstallHandlers(r *router.Router, mc router.MiddlewareChain) {
	mc = mc.Extend(func(ctx *router.Context, next router.Handler) {
		context, err := checkAccess(ctx.Context, ctx.HandlerPath, nil)
		ctx.Context = context
		if err != nil {
			logging.Errorf(ctx.Context, "Failed authorization %v", err)
			return
		}
		next(ctx)
	})
	r.POST("/pubsub/hart", mc, HaRTPushHandler)
}

// checkAccess verifies that the request is from an authorized user.
func checkAccess(ctx context.Context, rpcName string, _ proto.Message) (context.Context, error) {
	logging.Debugf(ctx, "Check access for %s", rpcName)
	group := []string{"mdb/chrome-fleet-software-team", "mdb/chrome-labs", "mdb/hwops-nsi", "mdb/chromeos-labs", "mdb/chromeos-labs-tvcs", "mdb/acs-labs"}
	if strings.HasPrefix(rpcName, "Import") {
		group = []string{"mdb/chrome-fleet-software-team"}
	}
	if strings.HasPrefix(rpcName, "List") || strings.HasPrefix(rpcName, "Get") || strings.HasPrefix(rpcName, "BatchGet") {
		group = []string{"mdb/chrome-labs", "mdb/chrome-fleet-software-team", "machine-db-readers", "mdb/chromeos-labs", "mdb/chromeos-labs-tvcs", "mdb/acs-labs", "chromeos-inventory-readonly-access"}
	}
	switch rpcName {
	case "CreateMachineLSE", "UpdateMachineLSE", "CreateVM", "UpdateVM", "UpdateMachineLSEDeployment", "BatchUpdateMachineLSEDeployment", "CreateAsset", "UpdateAsset", "RackRegistration", "UpdateRack", "CreateSchedulingUnit", "UpdateSchedulingUnit", "UpdateConfigBundle":
		group = []string{"mdb/chrome-labs", "mdb/chrome-fleet-software-team", "chromeos-inventory-setup-label-write-access", "machine-db-writers", "chromeos-inventory-status-label-write-access"}
	case "DeleteMachineLSE", "CreateVlan", "UpdateVlan", "DeleteVlan", "DeleteVM", "DeleteSchedulingUnit":
		group = []string{"mdb/chrome-labs", "mdb/chrome-fleet-software-team", "chromeos-inventory-privileged-access"}
	case "DeleteMachine":
		group = append(group, "mdb/hwops-nsi", "chromeos-inventory-privileged-access")
	case "GetMachine", "GetState", "GetCachingService", "GetChromeOSDeviceData", "GetMachineLSE", "GetSchedulingUnit":
		group = append(group, "chromeos-inventory-readonly-access", "machine-db-readers")
	case "UpdateState", "UpdateDutState":
		group = append(group, "chromeos-inventory-status-label-write-access")
	case "/pubsub/hart":
		//TODO(anushruth): Rename group to UFS-pubsub-push-access after removing functionality from IV2
		group = append(group, "chromeos-inventory-pubsub-push-access")
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
