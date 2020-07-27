package dumper

import (
	"context"

	"cloud.google.com/go/datastore"
	"go.chromium.org/luci/common/logging"

	iv2ds "infra/libs/cros/lab_inventory/datastore"
	iv2pr "infra/libs/fleet/protos"
	iv2pr2 "infra/libs/fleet/protos/go"
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

// GetAllAssetInfo retrieves all the asset info data from inventory-V2
func GetAllAssetInfo(ctx context.Context, client *datastore.Client) (map[string]*iv2pr2.AssetInfo, error) {
	var assetInfoEntities []*iv2ds.AssetInfoEntity

	_, err := client.GetAll(ctx, datastore.NewQuery(iv2ds.AssetInfoEntityKind), &assetInfoEntities)
	if err != nil {
		return nil, err
	}
	logging.Debugf(ctx, "Found %v assetInfoEntities", len(assetInfoEntities))

	assetInfos := make(map[string]*iv2pr2.AssetInfo, len(assetInfoEntities))
	for _, a := range assetInfoEntities {
		assetInfos[a.Info.GetAssetTag()] = &a.Info
	}
	return assetInfos, nil
}
