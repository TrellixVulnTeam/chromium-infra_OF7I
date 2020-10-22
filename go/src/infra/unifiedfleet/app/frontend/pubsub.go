package frontend

import (
	"net/http"
	"regexp"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"
	"google.golang.org/protobuf/testing/protocmp"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"
)

var macAddress = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:]){5}([0-9A-Fa-f]{2})$`)

// HaRTPushHandler handles the pubsub push responses from HaRT pubsub
//
// Decodes the response sent by PubSub and updates datastore. It doesn't
// return anything as required by https://cloud.google.com/pubsub/docs/push,
// this is because by default the return is 200 OK for http POST requests.
// It does return a http error if the datastore update fails.
func HaRTPushHandler(context *router.Context) {
	ctx := context.Context
	res, err := util.NewPSRequest(context.Request)
	if err != nil {
		logging.Errorf(ctx, "Failed to read push req %v", err)
		return
	}
	data, err := res.DecodeMessage()
	if err != nil {
		logging.Errorf(ctx, "Failed to read data %v", err)
		return
	}
	// Decode the proto contained in the payload
	var response ufspb.AssetInfoResponse
	perr := proto.Unmarshal(data, &response)
	if perr != nil {
		// Avoid returning error, as the data contains some assets not
		// known to HaRT and those will always fail.
		logging.Errorf(ctx, "Failed to decode proto %v", perr)
		return
	}
	if response.GetRequestStatus() == ufspb.RequestStatus_OK {
		allinfo := response.GetAssets()
		logging.Infof(ctx, "Updating %v assets", len(allinfo))
		assetsToUpdate := make([]*ufspb.Asset, 0, len(allinfo))
		for _, iv2assetinfo := range allinfo {
			ufsAsset, err := registration.GetAsset(ctx, iv2assetinfo.GetAssetTag())
			if err != nil {
				logging.Warningf(ctx, "Cannot update asset [%v], not found in DS", iv2assetinfo.GetAssetTag())
				continue
			}
			if !compare(ufsAsset.GetInfo(), iv2assetinfo) {
				logging.Infof(ctx, "Updating %v", ufsAsset.GetName())
				ufsAsset.Info = iv2assetinfo
				assetsToUpdate = append(assetsToUpdate, ufsAsset)
			}
		}
		if _, err = registration.BatchUpdateAssets(ctx, assetsToUpdate); err != nil {
			// Return http err if we fail to update.
			http.Error(context.Writer, "Internal server error", http.StatusInternalServerError)
			logging.Warningf(ctx, "Failed to update assets %v", err)
		}
	}
	logging.Infof(ctx, "Status: %v", response.GetRequestStatus())
	missing := response.GetMissingAssetTags()
	logging.Infof(ctx, "Missing[%v]: %v", len(missing), missing)
	failed := response.GetFailedAssetTags()
	logging.Infof(ctx, "Failed[%v]: %v", len(failed), failed)
	logging.Infof(ctx, "Success reported for %v assets", len(response.GetAssets()))
	return
}

func compare(iv2assetinfo, ufsassetinfo *ufspb.AssetInfo) bool {
	return cmp.Equal(iv2assetinfo, ufsassetinfo, protocmp.Transform())
}
