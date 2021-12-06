// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"infra/unifiedfleet/app/external"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/config/go/test/api"

	ufspb "infra/unifiedfleet/api/v1/models"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
)

func parseDutAttribute(t *testing.T, protoText string) api.DutAttribute {
	var da api.DutAttribute
	if err := jsonpb.UnmarshalString(protoText, &da); err != nil {
		t.Fatalf("Error unmarshalling example text: %s", err)
	}
	return da
}

func mockMachineLSEWithLabConfigs(name string) *ufspb.MachineLSE {
	return &ufspb.MachineLSE{
		Name:     name,
		Hostname: name,
		Lse: &ufspb.MachineLSE_ChromeosMachineLse{
			ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
				ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
					DeviceLse: &ufspb.ChromeOSDeviceLSE{
						Device: &ufspb.ChromeOSDeviceLSE_Dut{
							Dut: &chromeosLab.DeviceUnderTest{
								Hostname: name,
								Peripherals: &chromeosLab.Peripherals{
									Audio: &chromeosLab.Audio{
										AudioBox: true,
									},
								},
								Licenses: []*chromeosLab.License{
									{
										Type:       chromeosLab.LicenseType_LICENSE_TYPE_WINDOWS_10_PRO,
										Identifier: "test-license",
									},
									{
										Type:       chromeosLab.LicenseType_LICENSE_TYPE_MS_OFFICE_STANDARD,
										Identifier: "test-license-2",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestConvert(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	ctx = external.WithTestingContext(ctx)
	ctx = useTestingCfg(ctx)

	t.Run("convert lab config label - happy path; single boolean value", func(t *testing.T) {
		dutMachinelse := mockMachineLSEWithLabConfigs("lse-1")
		dutState := mockDutState("dutstate-id-1", "dutstate-hostname-1")
		daText := `{
			"id": {
				"value": "peripheral-audio-box"
			},
			"aliases": [
				"label-audio_box"
			],
			"tleSource": {}
		}`
		da := parseDutAttribute(t, daText)
		want := []string{
			"peripheral-audio-box:true",
			"label-audio_box:true",
		}
		got, err := Convert(ctx, &da, nil, dutMachinelse, dutState)
		if err != nil {
			t.Fatalf("Convert failed: %s", err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Convert returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("convert lab config label - happy path; array of values", func(t *testing.T) {
		dutMachinelse := mockMachineLSEWithLabConfigs("lse-1")
		dutState := mockDutState("dutstate-id-1", "dutstate-hostname-1")
		daText := `{
			"id": {
				"value": "misc-license"
			},
			"aliases": [
				"label-license"
			],
			"tleSource": {}
		}`
		da := parseDutAttribute(t, daText)
		want := []string{
			"misc-license:LICENSE_TYPE_WINDOWS_10_PRO,LICENSE_TYPE_MS_OFFICE_STANDARD",
			"label-license:LICENSE_TYPE_WINDOWS_10_PRO,LICENSE_TYPE_MS_OFFICE_STANDARD",
		}
		got, err := Convert(ctx, &da, nil, dutMachinelse, dutState)
		if err != nil {
			t.Fatalf("Convert failed: %s", err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Convert returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("convert dut state label - happy path; single value", func(t *testing.T) {
		dutState := mockDutState("dutstate-id-1", "dutstate-hostname-1")
		dutState.WorkingBluetoothBtpeer = 10
		daText := `{
			"id": {
				"value": "peripheral-num-btpeer"
			},
			"aliases": [
				"label-working_bluetooth_btpeer"
			],
			"tleSource": {}
		}`
		da := parseDutAttribute(t, daText)
		want := []string{
			"peripheral-num-btpeer:10",
			"label-working_bluetooth_btpeer:10",
		}
		got, err := Convert(ctx, &da, nil, nil, dutState)
		if err != nil {
			t.Fatalf("Convert failed: %s", err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Convert returned unexpected diff (-want +got):\n%s", diff)
		}
	})
}
