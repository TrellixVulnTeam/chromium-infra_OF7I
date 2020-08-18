package frontend

import (
	"regexp"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/controller"
	"infra/unifiedfleet/app/util"
)

var macAddress = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:]){5}([0-9A-Fa-f]{2})$`)

// HaRTPushHandler handles the pubsub push responses from HaRT pubsub
//
// Decodes the response sent by PubSub and updates datastore. It doesn't
// return anything as required by https://cloud.google.com/pubsub/docs/push,
// this is because by default the return is 200 OK for http POST requests.
// It does not return any 4xx codes on error because it could lead to a loop
// where PubSub tries to push same message again which is rejected.
func HaRTPushHandler(context *router.Context) {
	//TODO(anushruth): Setup HTTP error reporting
	res, err := util.NewPSRequest(context.Request)
	if err != nil {
		logging.Errorf(context.Context, "Failed to read push req %v", err)
		return
	}
	data, err := res.DecodeMessage()
	if err != nil {
		logging.Errorf(context.Context, "Failed to read data %v", err)
		return
	}
	// Decode the proto contained in the payload
	var response ufspb.AssetInfoResponse
	perr := proto.Unmarshal(data, &response)
	if perr == nil {
		if response.GetRequestStatus() == ufspb.RequestStatus_OK {
			ai := response.GetAssets()
			for _, asset := range ai {
				machine, err := controller.GetMachine(context.Context, asset.GetAssetTag())
				if err != nil {
					logging.Errorf(context.Context, "Machine [%v] not found", asset.GetAssetTag())
					continue
				}
				hostname := machine.GetLocation().GetBarcodeName()
				device := &ufspb.ChromeOSMachine{
					ReferenceBoard: asset.GetReferenceBoard(),
					BuildTarget:    asset.GetBuildTarget(),
					Model:          asset.GetModel(),
					GoogleCodeName: asset.GetGoogleCodeName(),
					MacAddress:     asset.GetEthernetMacAddress(),
					Sku:            asset.GetSku(),
					Phase:          asset.GetPhase(),
					CostCenter:     asset.GetCostCenter(),
				}
				if strings.Contains(hostname, "labstation") || strings.Contains(device.GoogleCodeName, "Labstation") {
					device.DeviceType = ufspb.ChromeOSDeviceType_DEVICE_LABSTATION
				} else if strings.Contains(device.GoogleCodeName, "Servo") || macAddress.MatchString(asset.GetAssetTag()) {
					device.DeviceType = ufspb.ChromeOSDeviceType_DEVICE_SERVO
				} else {
					// TODO(anushruth): Default cannot be chromebook, But currently
					// we don't have data to determine this.
					device.DeviceType = ufspb.ChromeOSDeviceType_DEVICE_CHROMEBOOK
				}
				machine.Device = &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: device,
				}
				machine.UpdateTime = ptypes.TimestampNow()
				_, err = controller.UpdateMachine(context.Context, machine, nil)
				if err != nil {
					logging.Errorf(context.Context, "Failed to update machine %v", machine.GetName())
				}
			}
		}
		logging.Infof(context.Context, "Status: %v", response.GetRequestStatus())
		missing := response.GetMissingAssetTags()
		logging.Infof(context.Context, "Missing[%v]: %v", len(missing), missing)
		failed := response.GetFailedAssetTags()
		logging.Infof(context.Context, "Failed[%v]: %v", len(failed), failed)
		logging.Infof(context.Context, "Success reported for %v assets", len(response.GetAssets()))
	} else {
		logging.Errorf(context.Context, "Failed to decode proto %v", perr)
	}
	return
}
