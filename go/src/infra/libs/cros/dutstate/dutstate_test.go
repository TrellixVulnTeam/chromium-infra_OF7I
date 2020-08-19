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
	"google.golang.org/protobuf/types/known/timestamppb"

	ufsProto "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
)

type FakeUFSClient struct {
	getStateMap    map[string]ufsProto.State
	getStateErr    error
	updateStateMap map[string]ufsProto.State
	updateStateErr error
}

func (c *FakeUFSClient) GetState(ctx context.Context, req *ufsAPI.GetStateRequest, opts ...grpc.CallOption) (*ufsProto.StateRecord, error) {
	if c.getStateErr == nil {
		return &ufsProto.StateRecord{
			ResourceName: req.GetResourceName(),
			State:        c.getStateMap[req.GetResourceName()],
			UpdateTime:   timestamppb.Now(),
		}, nil
	}
	return nil, c.getStateErr
}
func (c *FakeUFSClient) UpdateState(ctx context.Context, req *ufsAPI.UpdateStateRequest, opts ...grpc.CallOption) (*ufsProto.StateRecord, error) {
	if c.updateStateErr == nil {
		c.updateStateMap[req.State.GetResourceName()] = req.State.GetState()
		return &ufsProto.StateRecord{
			ResourceName: req.State.GetResourceName(),
			State:        req.State.GetState(),
			UpdateTime:   timestamppb.Now(),
		}, nil
	}
	return nil, c.updateStateErr
}

func TestReadState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	Convey("Read state from USF", t, func() {
		c := &FakeUFSClient{
			getStateMap: map[string]ufsProto.State{
				"hosts/host1": ufsProto.State_STATE_REPAIR_FAILED,
				"hosts/host2": ufsProto.State_STATE_DEPLOYED_TESTING,
			},
		}
		r := Read(ctx, c, "host1")
		So(r.State, ShouldEqual, "repair_failed")
		So(r.Time, ShouldNotEqual, 0)

		r = Read(ctx, c, "host2")
		So(r.State, ShouldEqual, "manual_repair")
		So(r.Time, ShouldNotEqual, 0)

		r = Read(ctx, c, "host3")
		So(r.State, ShouldEqual, "needs_repair")
		So(r.Time, ShouldNotEqual, 0)
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
			So(c.updateStateMap["hosts/host1"], ShouldEqual, ufsProto.State_STATE_REPAIR_FAILED)
		})

		Convey("set manual_repair and expect DEPLOYED_TESTING", func() {
			e := Update(ctx, c, "host2", "manual_repair")
			So(e, ShouldBeNil)
			So(c.updateStateMap, ShouldHaveLength, 1)
			So(c.updateStateMap["hosts/host2"], ShouldEqual, ufsProto.State_STATE_DEPLOYED_TESTING)
		})

		Convey("set incorrect state and expect UNSPECIFIED for UFS", func() {
			e := Update(ctx, c, "host2", "wrong_state")
			So(e, ShouldBeNil)
			So(c.updateStateMap, ShouldHaveLength, 1)
			So(c.updateStateMap["hosts/host2"], ShouldEqual, ufsProto.State_STATE_UNSPECIFIED)
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
			got := convertToUFSState(tc.in)
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
			State("needs_repair"),
		},
	}
	for _, tc := range testcases {
		tc := tc
		t.Run(tc.in.String(), func(t *testing.T) {
			t.Parallel()
			got := convertFromUFSState(tc.in)
			if diff := cmp.Diff(tc.out, got); diff != "" {
				t.Errorf("output mismatch (-want +got): %s\n", diff)
			}
		})
	}
}

func TestUFSResourceName(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		in  string
		out string
	}{
		{
			"My",
			"hosts/My",
		},
		{
			"host-/+01",
			"hosts/host-/+01",
		},
		{
			"chromeos6-row5-rack10-host6",
			"hosts/chromeos6-row5-rack10-host6",
		},
	}
	for _, tc := range testcases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got := makeUFSResourceName(tc.in)
			if diff := cmp.Diff(tc.out, got); diff != "" {
				t.Errorf("output mismatch (-want +got): %s\n", diff)
			}
		})
	}
}
