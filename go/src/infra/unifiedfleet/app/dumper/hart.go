package dumper

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.chromium.org/luci/common/logging"

	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"
)

// SyncAssetInfoFromHaRT publishes the request for asset info to HaRT.
//
// The response for this request will be made to an endpoint on an RPC call.
// This function only checks for the assets that have Device info missing or
// the last update on the device was 48 hours ago.
func SyncAssetInfoFromHaRT(ctx context.Context) error {
	// In UFS write to 'os' namespace
	var err error
	ctx, err = util.SetupDatastoreNamespace(ctx, util.OSNamespace)
	if err != nil {
		return err
	}

	res, err := registration.GetAllAssets(ctx)
	if err != nil {
		logging.Errorf(ctx, "Unable to get assets %v", err)
		return err
	}
	ids := make([]string, 0, len(res))
	for _, r := range res {
		// Request an update, if we don't have asset info or the last update
		// was more than 2 days ago. Filter out uuids, any name with " character in them and macs
		// Filtering out " because of how HaRT does their queries.
		if _, err := uuid.Parse(r.Name); err != nil && !strings.Contains(r.Name, `"`) && !macRegex.MatchString(r.Name) && (r.Info == nil || time.Since(r.UpdateTime.AsTime()).Hours() > 48.00) {
			ids = append(ids, r.Name)
		}
	}
	logging.Infof(ctx, "Updating %v devices", len(ids))
	return util.PublishHaRTAssetInfoRequest(ctx, ids)
}
