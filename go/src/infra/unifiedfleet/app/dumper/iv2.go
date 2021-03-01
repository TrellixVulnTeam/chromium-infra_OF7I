package dumper

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/datastore"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/go-cmp/cmp"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	iv2ds "infra/cros/lab_inventory/datastore"
	iv2pr "infra/libs/fleet/protos"
	iv2pr2 "infra/libs/fleet/protos/go"
	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/controller"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"
)

var macRegex = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:\.-]){5}([0-9A-Fa-f]{2})$`)

// List of fields to be ignored when comparing a machine object to another.
// Field names here should reflect *.proto not generated *.pb.go
var machineCmpIgnoreFields = []protoreflect.Name{
	protoreflect.Name("update_time"),
}

// List of fields to be ignored when comparing a Asset object to another.
// Field names here should reflect *.proto not generated *.pb.go
var assetCmpIgnoreFields = []protoreflect.Name{
	protoreflect.Name("update_time"),
	// Don't care about info, We can update it from HaRT directly
	protoreflect.Name("info"),
}

// SyncAssetsFromIV2 updates assets table in UFS using data from IV2
func SyncAssetsFromIV2(ctx context.Context) error {
	// UFS migration done, skip this job.
	if config.Get(ctx).GetDisableInv2Sync() {
		logging.Infof(ctx, "UFS migration done, skipping the InvV2 to UFS Assets sync")
		return nil
	}
	logging.Infof(ctx, "SyncAssetsFromIV2: InvV2 to UFS Assets sync")
	ut := ptypes.TimestampNow()
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

	// In UFS write to 'os' namespace
	ctx, err = util.SetupDatastoreNamespace(ctx, util.OSNamespace)
	if err != nil {
		return err
	}

	// Add all assets from inventory V2's asset table
	assetsToUpdate := make([]*ufspb.Asset, 0, len(assets))
	for _, asset := range assets {
		var iv2Asset *ufspb.Asset
		ufsAsset, err := registration.GetAsset(ctx, asset.GetId())
		iv2Asset, convErr := createAssetsFromChopsAsset(asset, assetInfos[asset.GetId()], assetsToHostname[asset.GetId()])
		if convErr != nil {
			logging.Warningf(ctx, "Unable to create asset %v: %v", asset, convErr)
			continue
		}
		if err != nil {
			// Create rack when creating assets if rack is missing
			if err := checkRackExists(ctx, iv2Asset.GetLocation().GetRack()); err != nil {
				if err := registerRacksForAsset(ctx, iv2Asset); err != nil {
					logging.Warningf(ctx, "Unable to create rack %s (asset %s): %s", iv2Asset.GetLocation().GetRack(), iv2Asset.GetName(), err.Error())
					continue
				}
			}
			iv2Asset.UpdateTime = ut
			_, err := controller.AssetRegistration(ctx, iv2Asset)
			if err != nil {
				logging.Warningf(ctx, "Failed to register asset %v: %v", asset, err)
			}
			continue
		}
		if !Cmp(iv2Asset, ufsAsset) {
			// Avoid updating assetinfo from IV2. It will be updated directly
			iv2Asset.Info = ufsAsset.Info
			iv2Asset.UpdateTime = ut
			assetsToUpdate = append(assetsToUpdate, iv2Asset)
		}
	}
	logging.Infof(ctx, "Updating: %v", assetsToUpdate)
	_, err = registration.BatchUpdateAssets(ctx, assetsToUpdate)
	if err != nil {
		return err
	}

	// Reference all assets based on inventory V2's device (lab config) table
	return updateAssetsFromInventoryV2(ctx)
}

func updateAssetsFromInventoryV2(ctx context.Context) error {
	t, err := auth.GetRPCTransport(ctx, auth.AsSelf)
	if err != nil {
		return err
	}
	inv2Client := invV2Api.NewInventoryPRPCClient(&prpc.Client{
		C:    &http.Client{Transport: t},
		Host: config.Get(ctx).CrosInventoryHost,
	})
	resp, err := inv2Client.ListCrosDevicesLabConfig(ctx, &invV2Api.ListCrosDevicesLabConfigRequest{})
	if err != nil {
		return err
	}

	assets, err := registration.GetAllAssets(ctx)
	if err != nil {
		return err
	}
	existingAssetMap := make(map[string]*ufspb.Asset, 0)
	for _, a := range assets {
		existingAssetMap[a.GetName()] = a
	}
	assetsToUpdate := util.ToOSAssets(resp.GetLabConfigs(), existingAssetMap)

	logging.Infof(ctx, "UFS already contains %d assets", len(assets))
	logging.Infof(ctx, "Inventory V2 contains %d machines", len(resp.GetLabConfigs()))
	logging.Infof(ctx, "Updating %d assets based on inventory", len(assetsToUpdate))

	pageSize := 500
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(assetsToUpdate))
		_, err = registration.BatchUpdateAssets(ctx, assetsToUpdate[i:end])
		if err != nil {
			return err
		}
		if i+pageSize >= len(assetsToUpdate) {
			break
		}
	}
	logging.Infof(ctx, "Successfully updated %d assets", len(assetsToUpdate))
	return nil
}

func checkRackExists(ctx context.Context, rack string) error {
	// It's possible that an asset's rack is empty because
	// a. we cannot parse rack from its hostname, e.g. chromeos1-...jetstream-host5
	// b. the asset is not scanned/doesn't exist in HaRT
	if rack == "" {
		return nil
	}
	return controller.ResourceExist(ctx, []*controller.Resource{controller.GetRackResource(rack)}, nil)
}

func registerRacksForAsset(ctx context.Context, asset *ufspb.Asset) error {
	l := asset.GetLocation()
	rack := &ufspb.Rack{
		Name: l.GetRack(),
		Location: &ufspb.Location{
			Aisle:       l.GetAisle(),
			Row:         l.GetRow(),
			Rack:        l.GetRack(),
			RackNumber:  l.GetRackNumber(),
			BarcodeName: l.GetRack(),
			Zone:        l.GetZone(),
		},
		Description:   "Added from IV2 by SyncAssetsFromIV2",
		ResourceState: ufspb.State_STATE_SERVING,
		Realm:         util.ToUFSRealm(l.GetZone().String()),
	}
	logging.Infof(ctx, "Add rack: %v", rack)
	_, err := controller.RackRegistration(ctx, rack)
	return err
}

// SyncMachinesFromAssets updates machines table from assets table
//
// Checks all the DUT and Labstation assets and creates/updates machines if required.
func SyncMachinesFromAssets(ctx context.Context) error {
	// In UFS write to 'os' namespace
	var err error
	ctx, err = util.SetupDatastoreNamespace(ctx, util.OSNamespace)
	if err != nil {
		return err
	}
	logging.Infof(ctx, "SyncMachinesFromAssets")
	assets, err := registration.GetAllAssets(ctx)
	if err != nil {
		return err
	}
	for _, asset := range assets {
		// Store DUTs and Labstations as machines
		if asset.GetType() == ufspb.AssetType_DUT || asset.GetType() == ufspb.AssetType_LABSTATION {
			// Create rack when creating machines
			if err := checkRackExists(ctx, asset.GetLocation().GetRack()); err != nil {
				if err := registerRacksForAsset(ctx, asset); err != nil {
					logging.Warningf(ctx, "Unable to create rack %s: %s", asset.GetLocation().GetRack(), err.Error())
					continue
				}
			}
			aMachine := controller.CreateMachineFromAsset(asset)
			if aMachine == nil {
				continue
			}
			ufsMachine, err := controller.GetMachine(ctx, asset.GetName())
			if err != nil && util.IsNotFoundError(err) {
				// Create a new machine
				_, err := controller.MachineRegistration(ctx, aMachine)
				if err != nil {
					logging.Warningf(ctx, "Unable to create machine %v %v", aMachine, err)
				}
			} else if ufsMachine != nil && !Compare(aMachine, ufsMachine) {
				// Serial number, Hwid, Sku of UFS machine is updated by SSW in
				// UpdateDutMeta https://source.corp.google.com/chromium_infra/go/src/infra/unifiedfleet/app/controller/machine.go;l=182
				// Dont rely on Hart for Serial number, Hwid, Sku and
				// macaddress. Copy back original values.
				aMachine.SerialNumber = ufsMachine.GetSerialNumber()
				aMachine.GetChromeosMachine().Hwid = ufsMachine.GetChromeosMachine().GetHwid()
				aMachine.GetChromeosMachine().Sku = ufsMachine.GetChromeosMachine().GetSku()
				aMachine.GetChromeosMachine().MacAddress = ufsMachine.GetChromeosMachine().GetMacAddress()
				_, err := controller.UpdateMachine(ctx, aMachine, nil)
				if err != nil {
					logging.Warningf(ctx, "Failed to update machine %v %v", aMachine, err)
				}
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

// Cmp does protobuf comparison between both inputs
func Cmp(iv2Asset, ufsAsset *ufspb.Asset) bool {
	opts1 := protocmp.IgnoreFields(ufsAsset, assetCmpIgnoreFields...)
	opts2 := protocmp.Transform()
	return cmp.Equal(iv2Asset, ufsAsset, opts1, opts2)
}

// createAssetsFromChopsAsset returns Asset proto constructed from ChopsAsset and AssetInfo proto
func createAssetsFromChopsAsset(asset *iv2pr.ChopsAsset, assetinfo *iv2pr2.AssetInfo, hostname string) (*ufspb.Asset, error) {
	a := &ufspb.Asset{
		Name: asset.GetId(),
		Location: &ufspb.Location{
			Aisle:       asset.GetLocation().GetAisle(),
			Row:         asset.GetLocation().GetRow(),
			Shelf:       asset.GetLocation().GetShelf(),
			Position:    asset.GetLocation().GetPosition(),
			BarcodeName: hostname,
		},
	}
	if assetinfo != nil {
		a.Info = &ufspb.AssetInfo{
			AssetTag:           assetinfo.GetAssetTag(),
			SerialNumber:       assetinfo.GetSerialNumber(),
			CostCenter:         assetinfo.GetCostCenter(),
			GoogleCodeName:     assetinfo.GetGoogleCodeName(),
			Model:              assetinfo.GetModel(),
			BuildTarget:        assetinfo.GetBuildTarget(),
			ReferenceBoard:     assetinfo.GetReferenceBoard(),
			EthernetMacAddress: assetinfo.GetEthernetMacAddress(),
			Sku:                assetinfo.GetSku(),
			Phase:              assetinfo.GetPhase(),
		}
	}

	a.Location.Zone = util.LabToZone(asset.GetLocation().GetLab())
	if a.Location.Zone == ufspb.Zone_ZONE_CROS_GOOGLER_DESK && hostname == "" {
		a.Location.BarcodeName = asset.GetLocation().GetLab()
	}
	// Construct rack name as `chromeos[$zone]`-row`$row`-rack`$rack`
	loc := asset.GetLocation()
	var r strings.Builder
	if loc.GetLab() == "" {
		return nil, errors.Reason("Cannot create an asset without zone").Err()
	}
	r.WriteString(loc.GetLab())
	if row := loc.GetRow(); row != "" {
		r.WriteString("-row")
		r.WriteString(row)
	}
	if rack := loc.GetRack(); rack != "" {
		r.WriteString("-rack")
		r.WriteString(rack)
		a.Location.RackNumber = rack
	} else {
		// Avoid setting Rack to zone name, e.g. chromeos2
		r.WriteString("-norack")
	}
	a.Location.Rack = r.String()
	if assetinfo != nil && assetinfo.GetGoogleCodeName() != "" {
		// Convert the model to all lowercase for compatibility with rest of the data
		a.Model = strings.ToLower(assetinfo.GetGoogleCodeName())
	}
	// Device can be one of DUT, Labstation, Servo, etc,.
	if a.Model == "" {
		// Some servos are recorded using their ethernet mac address
		if macRegex.MatchString(a.GetName()) {
			a.Type = ufspb.AssetType_SERVO
		} else {
			a.Type = ufspb.AssetType_UNDEFINED
		}
	} else if strings.Contains(a.Model, "labstation") {
		a.Type = ufspb.AssetType_LABSTATION
	} else if strings.Contains(a.Model, "servo") {
		a.Type = ufspb.AssetType_SERVO
	} else {
		// The asset is a DUT if it has model info and isn't a labstation or servo.
		a.Type = ufspb.AssetType_DUT
	}
	return a, nil
}
