// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufs

import (
	"context"
	"net"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	ufspb "infra/unifiedfleet/api/v1/models"
	lab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
)

func TestGetDutTopology_single(t *testing.T) {
	t.Parallel()
	ctx, cf := context.WithCancel(context.Background())
	defer cf()
	s := &fakeServer{
		ChromeOSDeviceData: &ufspb.ChromeOSDeviceData{
			LabConfig: &ufspb.MachineLSE{
				Hostname: "mary",
				Lse: &ufspb.MachineLSE_ChromeosMachineLse{
					ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
						ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
							DeviceLse: &ufspb.ChromeOSDeviceLSE{
								Device: &ufspb.ChromeOSDeviceLSE_Dut{
									Dut: &lab.DeviceUnderTest{
										Peripherals: &lab.Peripherals{
											Audio: &lab.Audio{
												AudioBox: true,
												Atrus:    true,
											},
											Chameleon: &lab.Chameleon{
												AudioBoard:           true,
												ChameleonPeripherals: []lab.ChameleonType{lab.ChameleonType_CHAMELEON_TYPE_DP},
											},
											Servo: &lab.Servo{
												ServoHostname: "servo_host",
												ServoPort:     33,
											},
											Wifi: &lab.Wifi{
												Wificell:    true,
												AntennaConn: lab.Wifi_CONN_CONDUCTIVE,
											},
											Touch: &lab.Touch{
												Mimo: true,
											},
											Camerabox: true,
											CameraboxInfo: &lab.Camerabox{
												Facing: lab.Camerabox_FACING_FRONT,
											},
											Cable: []*lab.Cable{
												{
													Type: lab.CableType_CABLE_AUDIOJACK,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	c := newFakeClient(ctx, t, s)
	got, err := GetDutTopology(ctx, c, "alice")
	if err != nil {
		t.Fatal(err)
	}
	want := &labapi.DutTopology{
		Id: &labapi.DutTopology_Id{Value: "alice"},
		Duts: []*labapi.Dut{
			{
				Id: &labapi.Dut_Id{Value: "mary"},
				DutType: &labapi.Dut_Chromeos{
					Chromeos: &labapi.Dut_ChromeOS{
						Audio: &labapi.Audio{
							AudioBox: true,
							Atrus:    true,
						},
						Chameleon: &labapi.Chameleon{
							AudioBoard:  true,
							Peripherals: []labapi.Chameleon_Peripheral{labapi.Chameleon_DP},
						},
						Servo: &labapi.Servo{
							ServodAddress: &labapi.IpEndpoint{
								Address: "servo_host",
								Port:    33,
							},
						},
						Ssh: &labapi.IpEndpoint{
							Address: "mary",
							Port:    22,
						},
						Wifi: &labapi.Wifi{
							Environment: labapi.Wifi_WIFI_CELL,
							Antenna: &labapi.WifiAntenna{
								Connection: labapi.WifiAntenna_CONDUCTIVE,
							},
						},
						Touch: &labapi.Touch{
							Mimo: true,
						},
						Camerabox: &labapi.Camerabox{
							Facing: labapi.Camerabox_FRONT,
						},
						Cables: []*labapi.Cable{
							{
								Type: labapi.Cable_AUDIOJACK,
							},
						},
					},
				},
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GetDutTopology() mismatch (-want +got):\n%s", diff)
	}
}

type fakeServer struct {
	ufsapi.UnimplementedFleetServer
	ChromeOSDeviceData *ufspb.ChromeOSDeviceData
}

func (s *fakeServer) GetChromeOSDeviceData(ctx context.Context, in *ufsapi.GetChromeOSDeviceDataRequest) (*ufspb.ChromeOSDeviceData, error) {
	return proto.Clone(s.ChromeOSDeviceData).(*ufspb.ChromeOSDeviceData), nil
}

// Make a fake client for testing.
// Cancel the context to clean up the fake server and client.
func newFakeClient(ctx context.Context, t *testing.T, s ufsapi.FleetServer) ufsapi.FleetClient {
	gs := grpc.NewServer()
	ufsapi.RegisterFleetServer(gs, s)
	l := bufconn.Listen(4096)
	go gs.Serve(l)
	go func() {
		<-ctx.Done()
		// This also closes the listener.
		gs.Stop()
	}()
	conn, err := grpc.DialContext(ctx, "", grpc.WithInsecure(),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return l.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		<-ctx.Done()
		conn.Close()
	}()
	return ufsapi.NewFleetClient(conn)
}
