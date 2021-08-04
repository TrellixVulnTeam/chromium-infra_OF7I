// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.package utils

package dutstate

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	ufsProto "infra/unifiedfleet/api/v1/models"
	ufslab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
)

type FakeUFSClient struct {
	getStateMap    map[string]ufsProto.State
	getStateErr    error
	updateStateMap map[string]ufsProto.State
	updateStateErr error
}

func TestReadState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	Convey("Read state from USF", t, func() {
		c := &FakeUFSClient{
			getStateMap: map[string]ufsProto.State{
				"machineLSEs/host1": ufsProto.State_STATE_REPAIR_FAILED,
				"machineLSEs/host2": ufsProto.State_STATE_DEPLOYED_TESTING,
			},
		}
		r := Read(ctx, c, "host1")
		So(r.State, ShouldEqual, "repair_failed")
		So(r.Time, ShouldNotEqual, 0)

		r = Read(ctx, c, "host2")
		So(r.State, ShouldEqual, "manual_repair")
		So(r.Time, ShouldNotEqual, 0)

		r = Read(ctx, c, "not_found")
		So(r.State, ShouldEqual, "unknown")
		So(r.Time, ShouldEqual, 0)

		r = Read(ctx, c, "fail")
		So(r.State, ShouldEqual, "unknown")
		So(r.Time, ShouldEqual, 0)
	})
}

func TestUpdateState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	Convey("Read state from USF", t, func() {
		c := &FakeUFSClient{
			updateStateMap: map[string]ufsProto.State{},
		}

		Convey("set repair_failed and expect REPAIR_FAILED", func() {
			e := Update(ctx, c, "host1", "repair_failed")
			So(e, ShouldBeNil)
			So(c.updateStateMap, ShouldHaveLength, 1)
			So(c.updateStateMap["machineLSEs/host1"], ShouldEqual, ufsProto.State_STATE_REPAIR_FAILED)
		})

		Convey("set manual_repair and expect DEPLOYED_TESTING", func() {
			e := Update(ctx, c, "host2", "manual_repair")
			So(e, ShouldBeNil)
			So(c.updateStateMap, ShouldHaveLength, 1)
			So(c.updateStateMap["machineLSEs/host2"], ShouldEqual, ufsProto.State_STATE_DEPLOYED_TESTING)
		})

		Convey("set incorrect state and expect UNSPECIFIED for UFS", func() {
			e := Update(ctx, c, "host2", "wrong_state")
			So(e, ShouldBeNil)
			So(c.updateStateMap, ShouldHaveLength, 1)
			So(c.updateStateMap["machineLSEs/host2"], ShouldEqual, ufsProto.State_STATE_UNSPECIFIED)
		})
	})
}

func TestConvertToUFSState(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		in  State
		out ufsProto.State
	}{
		{
			State("ready"),
			ufsProto.State_STATE_SERVING,
		},
		{
			State("repair_failed"),
			ufsProto.State_STATE_REPAIR_FAILED,
		},
		{
			State("Ready "),
			ufsProto.State_STATE_UNSPECIFIED,
		},
	}
	for _, tc := range testcases {
		tc := tc
		t.Run(string(tc.in), func(t *testing.T) {
			t.Parallel()
			got := ConvertToUFSState(tc.in)
			if diff := cmp.Diff(tc.out, got); diff != "" {
				t.Errorf("output mismatch (-want +got): %s\n", diff)
			}
		})
	}
}

func TestConvertFromUFSState(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		in  ufsProto.State
		out State
	}{
		{
			ufsProto.State_STATE_SERVING,
			State("ready"),
		},
		{
			ufsProto.State_STATE_DEPLOYED_PRE_SERVING,
			State("needs_deploy"),
		},
		{
			ufsProto.State_STATE_UNSPECIFIED,
			State("unknown"),
		},
	}
	for _, tc := range testcases {
		tc := tc
		t.Run(tc.in.String(), func(t *testing.T) {
			t.Parallel()
			got := ConvertFromUFSState(tc.in)
			if diff := cmp.Diff(tc.out, got); diff != "" {
				t.Errorf("output mismatch (-want +got): %s\n", diff)
			}
		})
	}
}

func TestStateString(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		in  State
		out string
	}{
		{
			Ready,
			"ready",
		},
		{
			NeedsRepair,
			"needs_repair",
		},
		{
			NeedsReset,
			"needs_reset",
		},
		{
			Reserved,
			"reserved",
		},
		{
			State("Some custom"),
			"Some custom",
		},
	}
	for _, tc := range testcases {
		tc := tc
		t.Run(tc.in.String(), func(t *testing.T) {
			t.Parallel()
			got := tc.in.String()
			if diff := cmp.Diff(tc.out, got); diff != "" {
				t.Errorf("output mismatch (-want +got): %s\n", diff)
			}
		})
	}
}

func (c *FakeUFSClient) GetMachineLSE(ctx context.Context, req *ufsAPI.GetMachineLSERequest, opts ...grpc.CallOption) (*ufsProto.MachineLSE, error) {
	if c.getStateErr == nil {
		if req.GetName() == "machineLSEs/fail" {
			return nil, status.Error(codes.Unknown, "Somthing else")
		}
		if req.GetName() == "machineLSEs/not_found" {
			return nil, status.Error(codes.NotFound, "not_found")
		}
		if req.GetName() == "machineLSEs/host1" {
			return &ufsProto.MachineLSE{
				Name:          req.GetName(),
				ResourceState: c.getStateMap[req.GetName()],
				UpdateTime:    timestamppb.Now(),
				Lse: &ufsProto.MachineLSE_ChromeosMachineLse{
					ChromeosMachineLse: &ufsProto.ChromeOSMachineLSE{
						ChromeosLse: &ufsProto.ChromeOSMachineLSE_DeviceLse{
							DeviceLse: &ufsProto.ChromeOSDeviceLSE{
								Device: &ufsProto.ChromeOSDeviceLSE_Dut{
									Dut: &ufslab.DeviceUnderTest{},
								},
							},
						},
					},
				},
			}, nil
		}
		if req.GetName() == "machineLSEs/host2" {
			return &ufsProto.MachineLSE{
				Name:          req.GetName(),
				ResourceState: c.getStateMap[req.GetName()],
				UpdateTime:    timestamppb.Now(),
				Lse: &ufsProto.MachineLSE_ChromeosMachineLse{
					ChromeosMachineLse: &ufsProto.ChromeOSMachineLSE{
						ChromeosLse: &ufsProto.ChromeOSMachineLSE_DeviceLse{
							DeviceLse: &ufsProto.ChromeOSDeviceLSE{
								Device: &ufsProto.ChromeOSDeviceLSE_Labstation{
									Labstation: &ufslab.Labstation{},
								},
							},
						},
					},
				},
			}, nil
		}
	}
	return nil, c.getStateErr
}

func (c *FakeUFSClient) UpdateMachineLSE(ctx context.Context, req *ufsAPI.UpdateMachineLSERequest, opts ...grpc.CallOption) (*ufsProto.MachineLSE, error) {
	if c.updateStateErr == nil {
		c.updateStateMap[req.GetMachineLSE().GetName()] = req.GetMachineLSE().GetResourceState()
		if req.GetMachineLSE().GetName() == "machineLSEs/host1" {
			return &ufsProto.MachineLSE{
				Name:          req.GetMachineLSE().GetName(),
				ResourceState: req.GetMachineLSE().GetResourceState(),
				UpdateTime:    timestamppb.Now(),
				Lse: &ufsProto.MachineLSE_ChromeosMachineLse{
					ChromeosMachineLse: &ufsProto.ChromeOSMachineLSE{
						ChromeosLse: &ufsProto.ChromeOSMachineLSE_DeviceLse{
							DeviceLse: &ufsProto.ChromeOSDeviceLSE{
								Device: &ufsProto.ChromeOSDeviceLSE_Dut{
									Dut: &ufslab.DeviceUnderTest{},
								},
							},
						},
					},
				},
			}, nil
		}
		if req.GetMachineLSE().GetName() == "machineLSEs/host2" {
			return &ufsProto.MachineLSE{
				Name:          req.GetMachineLSE().GetName(),
				ResourceState: req.GetMachineLSE().GetResourceState(),
				UpdateTime:    timestamppb.Now(),
				Lse: &ufsProto.MachineLSE_ChromeosMachineLse{
					ChromeosMachineLse: &ufsProto.ChromeOSMachineLSE{
						ChromeosLse: &ufsProto.ChromeOSMachineLSE_DeviceLse{
							DeviceLse: &ufsProto.ChromeOSDeviceLSE{
								Device: &ufsProto.ChromeOSDeviceLSE_Labstation{
									Labstation: &ufslab.Labstation{},
								},
							},
						},
					},
				},
			}, nil
		}
	}
	return nil, c.updateStateErr
}
