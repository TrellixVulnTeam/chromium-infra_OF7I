package dumper

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"go.chromium.org/luci/common/logging"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/config"
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
	conf := config.Get(ctx).GetHart()
	proj := conf.GetProject()
	topic := conf.GetTopic()
	client, err := pubsub.NewClient(ctx, proj)
	if err != nil {
		logging.Errorf(ctx, "Unable to create pubsub client %v", err)
		return err
	}
	top := client.Topic(topic)

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
	// Need to batch the requests as long requests time out on HaRT backend
	// TODO(anushruth): Get batchSize from config
	batchSize := 100
	for i := 0; i < len(ids); i += batchSize {
		var msg *ufspb.AssetInfoRequest
		if (i + batchSize) <= len(ids) {
			msg = &ufspb.AssetInfoRequest{
				AssetTags: ids[i : i+batchSize],
			}
		} else {
			msg = &ufspb.AssetInfoRequest{
				AssetTags: ids[i:],
			}
		}
		data, e := proto.Marshal(msg)
		if e != nil {
			logging.Errorf(ctx, "Failed to marshal message: %v", err)
			continue
		}
		result := top.Publish(ctx, &pubsub.Message{
			Data: data,
		})
		// Wait until the transaction is completed
		s, e := result.Get(ctx)
		if e != nil {
			logging.Errorf(ctx, "PubSub req %v failed: %v", s, e)
		}
	}
	return nil
}
