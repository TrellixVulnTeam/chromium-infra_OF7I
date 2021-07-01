package dumper

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	api "infra/unifiedfleet/api/v1/cron"
)

// InstallCronServices installs ...
func InstallCronServices(apiServer *prpc.Server) {
	apiServer.AccessControl = prpc.AllowOriginAll
	api.RegisterCronServer(apiServer, &api.DecoratedCron{
		Service: &CronServerImpl{},
		Prelude: checkCronAccess,
	})
}

func checkCronAccess(ctx context.Context, rpcName string, _ proto.Message) (context.Context, error) {
	logging.Debugf(ctx, "Check access for %s", rpcName)
	// Only we trigger the cron jobs.
	group := []string{"mdb/chrome-fleet-software-team"}
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
