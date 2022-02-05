package attached_device

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"

	ufsapi "infra/unifiedfleet/api/v1/rpc"
)

const attachedDeviceDataProto = `
lab_config: {
  hostname: "dummy_hostname"
  attached_device_lse: {
    os_version: {
      value: "dummy_value"
      description: "dummy_description"
      image: "dummy_image"
    }
    associated_hostname: "dummy_associated_hostname"
  }
}
machine: {
  serial_number: "1234567890"
  attached_device: {
    manufacturer: "dummy_manufacturer"
    device_type: ATTACHED_DEVICE_TYPE_ANDROID_PHONE
    build_target: "dummy_board"
    model: "dummy_model"
  }
}
`

var attachedDeviceLabels = []string{
	"associated_hostname:dummy_associated_hostname",
	"name:dummy_hostname",
	"serial_number:1234567890",
	"model:dummy_model",
	"board:dummy_board",
}

func TestConvertEmptyAttachedDeviceData(t *testing.T) {
	t.Parallel()
	data := ufsapi.AttachedDeviceData{}
	got := Convert(&data)
	var want []string
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Empty attached device labels mismatch (-want +got):\n%s", diff)
	}
}

func TestConvertAttachedDeviceData(t *testing.T) {
	t.Parallel()
	var data ufsapi.AttachedDeviceData
	if err := proto.UnmarshalText(attachedDeviceDataProto, &data); err != nil {
		t.Fatalf("Error unmarshalling example text: %s", err)
	}
	got := Convert(&data)
	if diff := cmp.Diff(attachedDeviceLabels, got); diff != "" {
		t.Errorf("Empty attached device labels mismatch (-want +got):\n%s", diff)
	}
}
