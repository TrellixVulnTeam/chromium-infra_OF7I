package dumper

import (
	"context"
	"regexp"
	"strings"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/datastore"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/go-cmp/cmp"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"

	iv2ds "infra/libs/cros/lab_inventory/datastore"
	iv2pr "infra/libs/fleet/protos"
	iv2pr2 "infra/libs/fleet/protos/go"
	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/controller"
)

var macAddress = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:]){5}([0-9A-Fa-f]{2})$`)

// List of fields to be ignored when comparing a machine object to another.
// Field names here should reflect *.proto not generated *.pb.go
var machineCmpIgnoreFields = []protoreflect.Name{
	protoreflect.Name("update_time"),
}

// SyncMachinesFromIV2 update machines table in UFS with data from IV2.
//
// Gathering data from Asset and AssetInfo entities in inventory DB to get
// location and ChromeOSMachine data. Also uses assets_in_swarming BQ table
// to estimate barcode_name.
func SyncMachinesFromIV2(ctx context.Context) error {
	logging.Infof(ctx, "SyncMachinesFromIV2")

	host := strings.TrimSuffix(config.Get(ctx).CrosInventoryHost, ".appspot.com")
	client, err := datastore.NewClient(ctx, host)
	if err != nil {
		return err
	}
	// BQ Client to get asset tag to hostname mapping
	bqClient := ctx.Value(contextKey).(*bigquery.Client)

	assets, err := GetAllAssets(ctx, client)
	if err != nil {
		return err
	}
	assetInfos, err := GetAllAssetInfo(ctx, client)
	if err != nil {
		return err
	}
	assetsToHostname, err := GetAssetToHostnameMap(ctx, bqClient)
	if err != nil {
		logging.Warningf(ctx, "Unable to get hostnames [%v], will"+
			"continue sync ignoring hostnames", err)
	}

	for _, asset := range assets {
		if assetInfos[asset.GetId()] != nil {
			iv2Machine := GetMachineFromAssets(asset,
				assetInfos[asset.GetId()], assetsToHostname[asset.GetId()])
			ufsMachine, err := controller.GetMachine(ctx, asset.GetId())
			if err == nil && assetsToHostname == nil && ufsMachine.Location != nil {
				// If we failed to read from BQ, machine hostname/barcode name is not known.
				// Copy the hostname from existing ufsMachine to avoid updating it.
				iv2Machine.Location.BarcodeName = ufsMachine.GetLocation().GetBarcodeName()
			}
			if err != nil && status.Code(err) == codes.NotFound && ufsMachine == nil {
				// Machine doesn't exist, create a new one
				logging.Debugf(ctx, "Adding %v [%v] to machines data",
					iv2Machine.Name, iv2Machine.Location.BarcodeName)
				controller.CreateMachine(ctx, iv2Machine)
				continue
			}
			if err == nil && !Compare(iv2Machine, ufsMachine) {
				// Machine exists, but was updated in IV2
				controller.UpdateMachine(ctx, iv2Machine)
				continue
			}
			if err != nil {
				logging.Warningf(ctx, "Error retriving machine: %v", err)
			}
		}
	}

	return nil
}

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

// Compare does protobuf comparison between both inputs
func Compare(iv2Machine, ufsMachine *ufspb.Machine) bool {
	// Ignoring fields not required for comparison
	opts1 := protocmp.IgnoreFields(iv2Machine, machineCmpIgnoreFields...)
	// See: https://developers.google.com/protocol-buffers/docs/reference/go/faq#deepequal
	opts2 := protocmp.Transform()
	return cmp.Equal(iv2Machine, ufsMachine, opts1, opts2)
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
