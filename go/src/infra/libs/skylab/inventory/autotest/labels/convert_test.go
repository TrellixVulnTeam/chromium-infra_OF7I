// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package labels

import (
	"fmt"
	"sort"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/kylelemons/godebug/pretty"

	"infra/libs/skylab/inventory"
)

const fullTextProto = `
variant: "somevariant"
test_coverage_hints {
  usb_detect: true
  use_lid: true
  test_usbprinting: true
  test_usbaudio: true
  test_hdmiaudio: true
  test_audiojack: true
  recovery_test: true
  meet_app: true
  hangout_app: true
  chromesign: true
  chaos_nightly: true
  chaos_dut: true
}
self_serve_pools: "poolval"
reference_design: "reef"
wifi_chip: "wireless_xxxx"
platform: "platformval"
phase: 4
peripherals: {
  wificell: true
  stylus: true
  servo: true
  servo_state: 3
  mimo: true
  huddly: true
  conductive: true
  chameleon_type: 3
  chameleon_type: 5
  chameleon: true
  camerabox: true
  audio_loopback_dongle: true
  audio_box: true
  audio_board: true
  router_802_11ax: true
  working_bluetooth_btpeer: 3
}
os_type: 2
model: "modelval"
sku: "skuval"
hwid_sku: "eve_IntelR_CoreTM_i7_7Y75_CPU_1_30GHz_16GB"
brand: "HOMH"
ec_type: 1
cts_cpu: 1
cts_cpu: 2
cts_abi: 1
cts_abi: 2
critical_pools: 2
critical_pools: 1
cr50_phase: 2
cr50_ro_keyid: "prod"
cr50_ro_version: "1.2.3"
cr50_rw_keyid: "0xde88588d"
cr50_rw_version: "9.8.7"
capabilities {
  webcam: true
  video_acceleration: 6
  video_acceleration: 8
  touchpad: true
  touchscreen: true
  fingerprint: true
  telephony: "telephonyval"
  storage: "storageval"
  power: "powerval"
  modem: "modemval"
  lucidsleep: true
  hotwording: true
  graphics: "graphicsval"
  internal_display: true
  gpu_family: "gpufamilyval"
  flashrom: true
  detachablebase: true
  carrier: 2
  bluetooth: true
  atrus: true
}
board: "boardval"
arc: true
`

var fullLabels = []string{
	"arc",
	"atrus",
	"audio_board",
	"audio_box",
	"audio_loopback_dongle",
	"bluetooth",
	"board:boardval",
	"brand-code:HOMH",
	"camerabox",
	"carrier:tmobile",
	"chameleon",
	"chameleon:dp_hdmi",
	"chameleon:hdmi",
	"chaos_dut",
	"chaos_nightly",
	"chromesign",
	"conductive:True",
	"cr50-ro-keyid:prod",
	"cr50-ro-version:1.2.3",
	"cr50-rw-keyid:0xde88588d",
	"cr50-rw-version:9.8.7",
	"cr50:pvt",
	"cts_abi_arm",
	"cts_abi_x86",
	"cts_cpu_arm",
	"cts_cpu_x86",
	"detachablebase",
	"device-sku:skuval",
	"ec:cros",
	"fingerprint",
	"flashrom",
	"gpu_family:gpufamilyval",
	"graphics:graphicsval",
	"hangout_app",
	"hotwording",
	"huddly",
	"hw_video_acc_enc_vp9",
	"hw_video_acc_enc_vp9_2",
	"internal_display",
	"lucidsleep",
	"meet_app",
	"mimo",
	"model:modelval",
	"modem:modemval",
	"os:cros",
	"phase:DVT2",
	"platform:platformval",
	"pool:bvt",
	"pool:cq",
	"pool:poolval",
	"power:powerval",
	"recovery_test",
	"reference_design:reef",
	"router_802_11ax",
	"servo",
	"servo_state:broken",
	"sku:eve_IntelR_CoreTM_i7_7Y75_CPU_1_30GHz_16GB",
	"storage:storageval",
	"stylus",
	"telephony:telephonyval",
	"test_audiojack",
	"test_hdmiaudio",
	"test_usbaudio",
	"test_usbprinting",
	"touchpad",
	"touchscreen",
	"usb_detect",
	"use_lid",
	"variant:somevariant",
	"webcam",
	"wifi_chip:wireless_xxxx",
	"wificell",
	"working_bluetooth_btpeer:3",
}

func TestConvertEmpty(t *testing.T) {
	t.Parallel()
	ls := inventory.SchedulableLabels{}
	got := Convert(&ls)
	if len(got) > 0 {
		t.Errorf("Got nonempty labels %#v", got)
	}
}

func TestConvertFull(t *testing.T) {
	t.Parallel()
	var ls inventory.SchedulableLabels
	if err := proto.UnmarshalText(fullTextProto, &ls); err != nil {
		t.Fatalf("Error unmarshalling example text: %s", err)
	}
	got := Convert(&ls)
	sort.Sort(sort.StringSlice(got))
	want := make([]string, len(fullLabels))
	copy(want, fullLabels)
	if diff := pretty.Compare(want, got); diff != "" {
		t.Errorf("labels differ -want +got, %s", diff)
	}
}

var servoStateConvertStateCases = []struct {
	stateValue   int32
	expectLabels []string
}{
	{0, []string{}},
	{1, []string{"servo_state:working"}},
	{2, []string{"servo_state:not_connected"}},
	{3, []string{"servo_state:broken"}},
	{4, []string{"servo_state:wrong_config"}},
	{5, []string{}}, //wrong value
}

func TestConvertServoStateWorking(t *testing.T) {
	for _, testCase := range servoStateConvertStateCases {
		t.Run("State value is "+string(testCase.stateValue), func(t *testing.T) {
			var ls inventory.SchedulableLabels
			protoText := fmt.Sprintf(`peripherals: { servo_state: %v}`, testCase.stateValue)
			if err := proto.UnmarshalText(protoText, &ls); err != nil {
				t.Fatalf("Error unmarshalling example text: %s", err)
			}
			want := testCase.expectLabels
			got := Convert(&ls)
			if diff := pretty.Compare(want, got); diff != "" {
				t.Errorf(
					"Convert servo_state %v got labels differ -want +got, %s",
					testCase.stateValue,
					diff)
			}
		})
	}
}

func TestRevertEmpty(t *testing.T) {
	t.Parallel()
	want := inventory.NewSchedulableLabels()
	got := Revert(nil)
	if diff := pretty.Compare(want, *got); diff != "" {
		t.Errorf("labels differ -want +got, %s", diff)
	}
}

func TestRevertServoStateWithWrongCase(t *testing.T) {
	t.Parallel()
	want := inventory.NewSchedulableLabels()
	*want.Peripherals.ServoState = inventory.PeripheralState_NOT_CONNECTED
	labels := []string{"servo_state:Not_Connected"}
	got := Revert(labels)
	if diff := pretty.Compare(want, *got); diff != "" {
		t.Errorf("labels differ -want +got, %s", diff)
	}
}

var servoStateRevertCaseTests = []struct {
	labelValue  string
	expectState inventory.PeripheralState
}{
	{"Something", inventory.PeripheralState_UNKNOWN},
	{"WorKing", inventory.PeripheralState_WORKING},
	{"working", inventory.PeripheralState_WORKING},
	{"WORKING", inventory.PeripheralState_WORKING},
	{"Not_Connected", inventory.PeripheralState_NOT_CONNECTED},
	{"noT_CONnected", inventory.PeripheralState_NOT_CONNECTED},
	{"BroKen", inventory.PeripheralState_BROKEN},
	{"BROKEN", inventory.PeripheralState_BROKEN},
	{"broken", inventory.PeripheralState_BROKEN},
	{"Wrong_config", inventory.PeripheralState_WRONG_CONFIG},
	{"WRONG_CONFIG", inventory.PeripheralState_WRONG_CONFIG},
}

func TestRevertServoStateWithWrongValue(t *testing.T) {
	for _, testCase := range servoStateRevertCaseTests {
		t.Run(testCase.labelValue, func(t *testing.T) {
			want := inventory.NewSchedulableLabels()
			*want.Peripherals.ServoState = testCase.expectState
			labels := []string{fmt.Sprintf("servo_state:%s", testCase.labelValue)}
			got := Revert(labels)
			if diff := pretty.Compare(want, *got); diff != "" {
				t.Errorf(
					"Revert servo_state from %v made labels differ -want +got, %s",
					testCase.labelValue,
					diff)
			}
		})
	}
}

func TestRevertFull(t *testing.T) {
	t.Parallel()
	var want inventory.SchedulableLabels
	if err := proto.UnmarshalText(fullTextProto, &want); err != nil {
		t.Fatalf("Error unmarshalling example text: %s", err)
	}
	labels := make([]string, len(fullLabels))
	copy(labels, fullLabels)
	got := Revert(labels)
	if diff := pretty.Compare(want, *got); diff != "" {
		t.Errorf("labels differ -want +got, %s", diff)
	}
}

const fullTextProtoSpecial = `
test_coverage_hints {
  usb_detect: true
  use_lid: true
  test_usbprinting: true
  test_usbaudio: true
  test_hdmiaudio: true
  test_audiojack: true
  recovery_test: true
  meet_app: true
  hangout_app: true
  chromesign: true
  chaos_nightly: true
  chaos_dut: true
}
self_serve_pools: "poolval"
reference_design: "reef"
wifi_chip: "wireless_xxxx"
platform: "platformval"
phase: 4
peripherals: {
  wificell: true
  stylus: true
  servo: true
  servo_state: 3
  mimo: true
  huddly: true
  conductive: true
  chameleon_type: 3
  chameleon_type: 5
  chameleon: true
  camerabox: true
  audio_loopback_dongle: true
  audio_box: true
  audio_board: true
  router_802_11ax: true
  working_bluetooth_btpeer: 3
}
os_type: 2
model: "modelval"
sku: "skuval"
hwid_sku: "eve_IntelR_CoreTM_i7_7Y75_CPU_1_30GHz_16GB"
brand: "HOMH"
ec_type: 1
cts_cpu: 1
cts_cpu: 2
cts_abi: 1
cts_abi: 2
critical_pools: 2
critical_pools: 1
cr50_phase: 2
cr50_ro_keyid: "prod"
cr50_ro_version: "1.2.3"
cr50_rw_keyid: "0xde88588d"
cr50_rw_version: "9.8.7"
capabilities {
  webcam: true
  video_acceleration: 6
  video_acceleration: 8
  touchpad: true
  touchscreen: true
  fingerprint: true
  telephony: "telephonyval"
  storage: "storageval"
  power: "powerval"
  modem: "modemval"
  lucidsleep: true
  hotwording: true
  graphics: "graphicsval"
  internal_display: true
  gpu_family: "gpufamilyval"
  flashrom: true
  detachablebase: true
  carrier: 2
  bluetooth: true
  atrus: true
}
board: "boardval"
arc: true
`

var fullLabelsSpecial = []string{
	"arc",
	"atrus",
	"audio_board",
	"audio_box",
	"audio_loopback_dongle",
	"bluetooth",
	"board:boardval",
	"brand-code:HOMH",
	"camerabox",
	"carrier:tmobile",
	"chameleon",
	"chameleon:dp_hdmi",
	"chameleon:hdmi",
	"chaos_dut",
	"chaos_nightly",
	"chromesign",
	"conductive:True",
	"cr50-ro-keyid:prod",
	"cr50-ro-version:1.2.3",
	"cr50-rw-keyid:0xde88588d",
	"cr50-rw-version:9.8.7",
	"cr50:pvt",
	"cts_abi_arm",
	"cts_abi_x86",
	"cts_cpu_arm",
	"cts_cpu_x86",
	"detachablebase",
	"device-sku:skuval",
	"ec:cros",
	"fingerprint",
	"flashrom",
	"gpu_family:gpufamilyval",
	"graphics:graphicsval",
	"hangout_app",
	"hotwording",
	"huddly",
	"hw_video_acc_enc_vp9",
	"hw_video_acc_enc_vp9_2",
	"internal_display",
	"lucidsleep",
	"meet_app",
	"mimo",
	"model:modelval",
	"modem:modemval",
	"os:cros",
	"phase:DVT2",
	"platform:platformval",
	"pool:bvt",
	"pool:cq",
	"pool:poolval",
	"power:powerval",
	"recovery_test",
	"reference_design:reef",
	"router_802_11ax",
	"servo",
	"servo_state:broken",
	"sku:eve_IntelR_CoreTM_i7_7Y75_CPU_1_30GHz_16GB",
	"storage:storageval",
	"stylus",
	"telephony:telephonyval",
	"test_audiojack",
	"test_hdmiaudio",
	"test_usbaudio",
	"test_usbprinting",
	"touchpad",
	"touchscreen",
	"usb_detect",
	"use_lid",
	"variant:",
	"webcam",
	"wifi_chip:wireless_xxxx",
	"wificell",
	"working_bluetooth_btpeer:3",
}

// Test the special cases in revert, including
// * empty variant
func TestRevertSpecial(t *testing.T) {
	t.Parallel()
	var want inventory.SchedulableLabels
	if err := proto.UnmarshalText(fullTextProtoSpecial, &want); err != nil {
		t.Fatalf("Error unmarshalling example text: %s", err)
	}
	labels := make([]string, len(fullLabelsSpecial))
	copy(labels, fullLabelsSpecial)
	got := Revert(labels)
	if diff := pretty.Compare(want, *got); diff != "" {
		t.Errorf("labels differ -want +got, %s", diff)
	}
}
