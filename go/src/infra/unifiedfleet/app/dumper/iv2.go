package dumper

import (
	"context"
	"regexp"
	"strings"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/datastore"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/api/iterator"

	iv2ds "infra/libs/cros/lab_inventory/datastore"
	iv2pr "infra/libs/fleet/protos"
	iv2pr2 "infra/libs/fleet/protos/go"
	ufspb "infra/unifiedfleet/api/v1/proto"
)

var macAddress = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:]){5}([0-9A-Fa-f]{2})$`)

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

// GetMachineFromAssets returns Machine proto constructed from ChopsAsset
// and AssetInfo
//
// Machine object is constructed by directly assigning most of the data. Lab in
// Location is converted from string to enum using a switch case and
// hostname is used for BarcodeName. If hostname is empty, Lab name is used for
// BarcodeName
func GetMachineFromAssets(asset *iv2pr.ChopsAsset, assetInfo *iv2pr2.AssetInfo, hostname string) *ufspb.Machine {
	device := &ufspb.ChromeOSMachine{
		ReferenceBoard: assetInfo.GetReferenceBoard(),
		BuildTarget:    assetInfo.GetBuildTarget(),
		Model:          assetInfo.GetModel(),
		GoogleCodeName: assetInfo.GetGoogleCodeName(),
		MacAddress:     assetInfo.GetEthernetMacAddress(),
		Sku:            assetInfo.GetSku(),
		Phase:          assetInfo.GetPhase(),
		CostCenter:     assetInfo.GetCostCenter(),
	}
	if strings.Contains(hostname, "labstation") ||
		strings.Contains(device.GoogleCodeName, "Labstation") {
		device.DeviceType = ufspb.ChromeOSDeviceType_DEVICE_LABSTATION
	} else if strings.Contains(device.GoogleCodeName, "Servo") ||
		macAddress.MatchString(asset.GetId()) {
		device.DeviceType = ufspb.ChromeOSDeviceType_DEVICE_SERVO
	} else {
		// TODO(anushruth): Default cannot be chromebook, But currently
		// we don't have data to determine this.
		device.DeviceType = ufspb.ChromeOSDeviceType_DEVICE_CHROMEBOOK
	}
	machine := &ufspb.Machine{
		Name:         asset.GetId(),
		SerialNumber: assetInfo.GetSerialNumber(),
		Location: &ufspb.Location{
			Aisle:       asset.GetLocation().GetAisle(),
			Row:         asset.GetLocation().GetRow(),
			Rack:        asset.GetLocation().GetRack(),
			Shelf:       asset.GetLocation().GetShelf(),
			Position:    asset.GetLocation().GetPosition(),
			BarcodeName: hostname,
		},
		Device: &ufspb.Machine_ChromeosMachine{
			ChromeosMachine: device,
		},
		UpdateTime: ptypes.TimestampNow(),
	}
	if hostname == "" {
		machine.Location.BarcodeName = asset.GetLocation().GetLab()
	}
	// Mapping: chromeos1 => Santiam; chromeos2 => Atlantis;
	// chromeos4 => Destiny; chromeos6 => Prometheus;
	// chromeos{3,5,7,9,15} => Linda Vista
	switch asset.GetLocation().GetLab() {
	case "chromeos1":
		machine.Location.Lab = ufspb.Lab_LAB_CHROMEOS_SANTIAM
	case "chromeos2":
		machine.Location.Lab = ufspb.Lab_LAB_CHROMEOS_ATLANTIS
	case "chromeos3":
		machine.Location.Lab = ufspb.Lab_LAB_CHROMEOS_LINDAVISTA
	case "chromeos4":
		machine.Location.Lab = ufspb.Lab_LAB_CHROMEOS_DESTINY
	case "chromeos5":
		machine.Location.Lab = ufspb.Lab_LAB_CHROMEOS_LINDAVISTA
	case "chromeos6":
		machine.Location.Lab = ufspb.Lab_LAB_CHROMEOS_PROMETHEUS
	case "chromeos7":
		machine.Location.Lab = ufspb.Lab_LAB_CHROMEOS_LINDAVISTA
	case "chromeos9":
		machine.Location.Lab = ufspb.Lab_LAB_CHROMEOS_LINDAVISTA
	case "chromeos15":
		machine.Location.Lab = ufspb.Lab_LAB_CHROMEOS_LINDAVISTA
	default:
		machine.Location.Lab = ufspb.Lab_LAB_UNSPECIFIED
	}

	return machine
}
