package pubsub

import (
	"context"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/router"

	"infra/appengine/cros/lab_inventory/app/config"
	"infra/libs/cros/lab_inventory/hart"
)

const (
	hartPushEndpoint = "/push-handlers/hart"
)

// InstallHandlers installs the handlers implemented by the pubsub package.
func InstallHandlers(r *router.Router, mwBase router.MiddlewareChain) {
	r.POST(hartPushEndpoint, mwBase, func(c *router.Context) {
		logging.Infof(c.Context, "PubSub push by %s", auth.CurrentIdentity(c.Context))
		if validate(c.Context, hartPushEndpoint) {
			hart.PushHandler(c.Context, c.Request)
		}
	})
}

func validate(ctx context.Context, handle string) bool {
	pusher := auth.CurrentIdentity(ctx)
	accessGroup := config.Get(ctx).GetPubsubPushers()
	group := accessGroup.GetValue()
	allow, err := auth.IsMember(ctx, group)
	if err != nil {
		logging.Warningf(ctx, "Unable to authenticate %v. %v",
			pusher, err)
		return false
	}
	if !allow {
		logging.Warningf(ctx, "PubSub push permission denied for %v",
			pusher)
		return false
	}
	return true
}
