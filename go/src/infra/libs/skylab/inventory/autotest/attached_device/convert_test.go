package attached_device

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"

	ufspb "infra/unifiedfleet/api/v1/models"
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
  serial_number: "12345AbcDE"
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
	"serial_number:12345AbcDE",
	"model:dummy_model",
	"board:dummy_board",
	"os:android",
}

func getEmptyAttachedDeviceMachine() *ufspb.Machine {
	return &ufspb.Machine{
		Device: &ufspb.Machine_AttachedDevice{
			AttachedDevice: &ufspb.AttachedDevice{},
		},
	}
}

func TestConvertEmptyAttachedDeviceData(t *testing.T) {
	t.Parallel()
	data := ufsapi.AttachedDeviceData{}
	got := Convert(&data)
	want := []string{"os:unknown"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Empty attached device labels mismatch (-want +got):\n%s", diff)
	}
}

func TestConvertFullAttachedDeviceData(t *testing.T) {
	t.Parallel()
	var data ufsapi.AttachedDeviceData
	if err := proto.UnmarshalText(attachedDeviceDataProto, &data); err != nil {
		t.Fatalf("Error unmarshalling example text: %s", err)
	}
	got := Convert(&data)
	if diff := cmp.Diff(attachedDeviceLabels, got); diff != "" {
		t.Errorf("Attached device labels mismatch (-want +got):\n%s", diff)
	}
}

func TestConvertOsTypeAndroidPhone(t *testing.T) {
	t.Parallel()
	data := ufsapi.AttachedDeviceData{
		Machine: getEmptyAttachedDeviceMachine(),
	}
	data.GetMachine().GetAttachedDevice().DeviceType = ufspb.AttachedDeviceType_ATTACHED_DEVICE_TYPE_ANDROID_PHONE
	got := Convert(&data)
	want := []string{"os:android"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Attached device labels mismatch (-want +got):\n%s", diff)
	}
}

func TestConvertOsTypeAndroidTablet(t *testing.T) {
	t.Parallel()
	data := ufsapi.AttachedDeviceData{
		Machine: getEmptyAttachedDeviceMachine(),
	}
	data.GetMachine().GetAttachedDevice().DeviceType = ufspb.AttachedDeviceType_ATTACHED_DEVICE_TYPE_ANDROID_TABLET
	got := Convert(&data)
	want := []string{"os:android"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Attached device labels mismatch (-want +got):\n%s", diff)
	}
}

func TestConvertOsTypeApplePhone(t *testing.T) {
	t.Parallel()
	data := ufsapi.AttachedDeviceData{
		Machine: getEmptyAttachedDeviceMachine(),
	}
	data.GetMachine().GetAttachedDevice().DeviceType = ufspb.AttachedDeviceType_ATTACHED_DEVICE_TYPE_APPLE_PHONE
	got := Convert(&data)
	want := []string{"os:ios"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Attached device labels mismatch (-want +got):\n%s", diff)
	}
}

func TestConvertOsTypeAppleTablet(t *testing.T) {
	t.Parallel()
	data := ufsapi.AttachedDeviceData{
		Machine: getEmptyAttachedDeviceMachine(),
	}
	data.GetMachine().GetAttachedDevice().DeviceType = ufspb.AttachedDeviceType_ATTACHED_DEVICE_TYPE_APPLE_TABLET
	got := Convert(&data)
	want := []string{"os:ios"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Attached device labels mismatch (-want +got):\n%s", diff)
	}
}
