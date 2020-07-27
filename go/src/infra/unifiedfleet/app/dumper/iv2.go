package dumper

import (
	"context"

	"cloud.google.com/go/datastore"
	"go.chromium.org/luci/common/logging"

	iv2ds "infra/libs/cros/lab_inventory/datastore"
	iv2pr "infra/libs/fleet/protos"
)

// GetAllAssets retrieves all the asset data from inventory-V2
func GetAllAssets(ctx context.Context, client *datastore.Client) ([]*iv2pr.ChopsAsset, error) {
	var assetEntities []*iv2ds.AssetEntity

	k, err := client.GetAll(ctx, datastore.NewQuery(iv2ds.AssetEntityName), &assetEntities)
	if err != nil {
		return nil, err
	}
	logging.Debugf(ctx, "Found %v assetEntities", len(assetEntities))

	assets := make([]*iv2pr.ChopsAsset, 0, len(assetEntities))
	for idx, a := range assetEntities {
		// Add key to the asset. GetAll doesn't update keys but
		// returns []keys in order
		a.ID = k[idx].Name
		asset, err := a.ToChopsAsset()
		if err != nil {
			logging.Warningf(ctx, "Unable to parse %v: %v", a.ID, err)
		}
		assets = append(assets, asset)
	}
	return assets, nil
}
