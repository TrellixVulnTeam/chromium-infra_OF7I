// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/luci/appengine/gaetesting"
	"google.golang.org/grpc"

	ufspb "infra/unifiedfleet/api/v1/models"
	ufschromeoslab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
	ufsutil "infra/unifiedfleet/app/util"
)

// Fakes for UFS tests START

var fakeDUT = &ufspb.MachineLSE{
	Name:     ufsutil.AddPrefix(ufsutil.MachineLSECollection, "test-dut"),
	Hostname: "test-dut",
	Machines: []string{"test-machine-dut"},
	Lse: &ufspb.MachineLSE_ChromeosMachineLse{
		ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
			ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
				DeviceLse: &ufspb.ChromeOSDeviceLSE{
					Device: &ufspb.ChromeOSDeviceLSE_Dut{
						Dut: &ufschromeoslab.DeviceUnderTest{
							Hostname: "test-dut",
							Licenses: []*ufschromeoslab.License{fakeWin10ProLicense},
						},
					},
				},
			},
		},
	},
	Zone:          "ZONE_CHROMEOS6",
	ResourceState: ufspb.State_STATE_REGISTERED,
}

var fakeDUTNoLicense = &ufspb.MachineLSE{
	Name:     ufsutil.AddPrefix(ufsutil.MachineLSECollection, "test-dut-2"),
	Hostname: "test-dut-2",
	Machines: []string{"test-machine-dut-2"},
	Lse: &ufspb.MachineLSE_ChromeosMachineLse{
		ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
			ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
				DeviceLse: &ufspb.ChromeOSDeviceLSE{
					Device: &ufspb.ChromeOSDeviceLSE_Dut{
						Dut: &ufschromeoslab.DeviceUnderTest{
							Hostname: "test-dut-2",
						},
					},
				},
			},
		},
	},
	Zone:          "ZONE_CHROMEOS6",
	ResourceState: ufspb.State_STATE_REGISTERED,
}

var fakeWin10ProLicense = &ufschromeoslab.License{
	Identifier: "test-windows-10-pro-license",
	Type:       ufschromeoslab.LicenseType_LICENSE_TYPE_WINDOWS_10_PRO,
}

var fakeMsOfficeLicense = &ufschromeoslab.License{
	Identifier: "test-ms-office-license",
	Type:       ufschromeoslab.LicenseType_LICENSE_TYPE_MS_OFFICE_STANDARD,
}

var fakeUnspecifiedLicense = &ufschromeoslab.License{
	Identifier: "test-unspecified-license",
	Type:       ufschromeoslab.LicenseType_LICENSE_TYPE_UNSPECIFIED,
}

type fakeUFSClient struct {
	ufsapi.FleetClient
}

// GetMachineLSE fakes the GetMachineLSE api from UFS.
func (ic *fakeUFSClient) GetMachineLSE(ctx context.Context, in *ufsapi.GetMachineLSERequest, opts ...grpc.CallOption) (*ufspb.MachineLSE, error) {
	if in.GetName() == ufsutil.AddPrefix(ufsutil.MachineLSECollection, "test-dut") {
		return proto.Clone(fakeDUT).(*ufspb.MachineLSE), nil
	}
	if in.GetName() == ufsutil.AddPrefix(ufsutil.MachineLSECollection, "test-dut-2") {
		return proto.Clone(fakeDUTNoLicense).(*ufspb.MachineLSE), nil
	}
	return nil, errors.New("No MachineLSE found")
}

// Fakes for UFS tests END

func testingContext() context.Context {
	c := gaetesting.TestingContextWithAppID("dev~infra-fleet-tlw")
	return c
}

func TestGetUFSDeviceLicenses(t *testing.T) {
	t.Parallel()
	ctx := testingContext()

	t.Run("get licenses of device", func(t *testing.T) {
		got, err := getUFSDeviceLicenses(ctx, &fakeUFSClient{}, "test-dut")
		if err != nil {
			t.Fatalf("getUFSDeviceLicenses(fakeClient) failed: %s", err)
		}
		want := []*tls.License{
			{
				Name: "test-windows-10-pro-license",
				Type: tls.License_WINDOWS_10_PRO,
			},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("getUFSDeviceLicenses(fakeClient) returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("get licenses of device with no licenses", func(t *testing.T) {
		got, err := getUFSDeviceLicenses(ctx, &fakeUFSClient{}, "test-dut-2")
		if err != nil {
			t.Fatalf("getUFSDeviceLicenses(fakeClient) failed: %s", err)
		}
		var want []*tls.License
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("getUFSDeviceLicenses(fakeClient) returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("get licenses of non existent device", func(t *testing.T) {
		const hostname = "ghost"
		got, err := getUFSDeviceLicenses(ctx, &fakeUFSClient{}, hostname)
		if err == nil {
			t.Errorf("getUFSDeviceLicenses(fakeClient) succeeded with non existent device %s", hostname)
		}
		var want []*tls.License
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("getUFSDeviceLicenses(fakeClient) returned unexpected diff (-want +got):\n%s", diff)
		}
	})
}
