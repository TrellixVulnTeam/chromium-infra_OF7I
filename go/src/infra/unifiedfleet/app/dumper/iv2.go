package dumper

import (
	"context"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/datastore"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/api/iterator"

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

// GetAssetToHostnameMap gets the asset tag to hostname mapping from
// assets_in_swarming BQ table
func GetAssetToHostnameMap(ctx context.Context, client *bigquery.Client) (map[string]string, error) {
	type mapping struct {
		AssetTag string
		HostName string
	}
	//TODO(anushruth): Get table name, dataset and project from config
	q := client.Query(`
		SELECT a_asset_tag AS AssetTag, s_host_name AS HostName FROM ` +
		"`cros-lab-inventory.inventory.assets_in_swarming`")
	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}
	// Read the first mapping as TotalRows is not populated until first
	// call to Next()
	var d mapping
	err = it.Next(&d)
	assetsToHostname := make(map[string]string, int(it.TotalRows))
	assetsToHostname[d.AssetTag] = d.HostName

	for {
		err := it.Next(&d)
		if err == iterator.Done {
			break
		}
		if err != nil {
			logging.Warningf(ctx, "Failed to read a row from BQ: %v", err)
		}
		assetsToHostname[d.AssetTag] = d.HostName
	}
	logging.Debugf(ctx, "Found hostnames for %v devices", len(assetsToHostname))
	return assetsToHostname, nil
}
