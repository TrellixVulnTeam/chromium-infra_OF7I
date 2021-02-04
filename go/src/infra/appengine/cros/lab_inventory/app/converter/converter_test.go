// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package converter

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/luci/common/errors"

	apibq "infra/appengine/cros/lab_inventory/api/bigquery"
	"infra/cros/lab_inventory/datastore"
)

var testDeviceToBQMsgsData = []struct {
	name     string
	in       *datastore.DeviceOpResult
	out      *apibq.LabInventory
	stateOut *apibq.StateConfigInventory
	isGood   bool
}{
	{
		"empty",
		nil,
		nil,
		nil,
		false,
	},
	{
		"just timestamp & ID",
		&datastore.DeviceOpResult{
			Entity: &datastore.DeviceEntity{
				ID:        "476528fa-29f9-4bf8-8f39-92dc6c291ca1",
				LabConfig: []byte{},
				Updated:   time.Unix(42, 67),
			},
		},
		&apibq.LabInventory{
			Id:          "476528fa-29f9-4bf8-8f39-92dc6c291ca1",
			UpdatedTime: timestampOrPanic(time.Unix(42, 67)),
			Device:      &lab.ChromeOSDevice{},
		},
		&apibq.StateConfigInventory{
			Id:          "476528fa-29f9-4bf8-8f39-92dc6c291ca1",
			UpdatedTime: timestampOrPanic(time.Unix(42, 67)),
			State:       &lab.DutState{},
		},
		true,
	},
	{
		"full device",
		&datastore.DeviceOpResult{
			Timestamp: time.Unix(42, 67),
			Data:      nil,
			Err:       nil,
			Entity: &datastore.DeviceEntity{
				ID:       datastore.DeviceEntityID("476528fa-29f9-4bf8-8f39-92dc6c291ca1"),
				Hostname: "ac60fddf-0e51-45a0-a8ad-4c291270e649",
				LabConfig: marshalOrPanic(&lab.ChromeOSDevice{
					Id:           &lab.ChromeOSDeviceID{Value: "476528fa-29f9-4bf8-8f39-92dc6c291ca1"},
					SerialNumber: "",
				}),
				DutState: marshalOrPanic(&lab.DutState{
					Id:    &lab.ChromeOSDeviceID{Value: "476528fa-29f9-4bf8-8f39-92dc6c291ca1"},
					Servo: lab.PeripheralState_BROKEN,
				}),
				Updated: time.Unix(42, 67),
				Parent:  nil,
			},
		},
		&apibq.LabInventory{
			UpdatedTime: timestampOrPanic(time.Unix(42, 67)),
			Device: &lab.ChromeOSDevice{
				Id:           &lab.ChromeOSDeviceID{Value: "476528fa-29f9-4bf8-8f39-92dc6c291ca1"},
				SerialNumber: "",
			},
			Id:       "476528fa-29f9-4bf8-8f39-92dc6c291ca1",
			Hostname: "ac60fddf-0e51-45a0-a8ad-4c291270e649",
		},
		&apibq.StateConfigInventory{
			UpdatedTime: timestampOrPanic(time.Unix(42, 67)),
			State: &lab.DutState{
				Id:    &lab.ChromeOSDeviceID{Value: "476528fa-29f9-4bf8-8f39-92dc6c291ca1"},
				Servo: lab.PeripheralState_BROKEN,
			},
			Id: "476528fa-29f9-4bf8-8f39-92dc6c291ca1",
		},
		true,
	},
}

func TestDeviceToBQMsgs(t *testing.T) {
	t.Parallel()
	for _, tt := range testDeviceToBQMsgsData {
		t.Run(tt.name, func(t *testing.T) {
			out, stateOut, err := DeviceToBQMsgs(tt.in)
			isGood := err == nil
			if isGood != tt.isGood {
				t.Errorf("error mismatch: expected (%v) got (%v) err was (%#v)", tt.isGood, isGood, err)
			}
			if diff := cmp.Diff(tt.out, out, protocmp.Transform()); diff != "" {
				t.Errorf("unexpected diff (%s)", diff)
			}
			if stateDiff := cmp.Diff(tt.stateOut, stateOut, protocmp.Transform()); stateDiff != "" {
				t.Errorf("unexpected diff (%s)", stateDiff)
			}
		})
	}
}

var testDeviceToBQMsgsSeqData = []struct {
	name     string
	in       datastore.DeviceOpResults
	out      []proto.Message
	stateOut []proto.Message
	isGood   bool
}{
	{
		"nil is failure",
		datastore.DeviceOpResults(nil),
		nil,
		nil,
		false,
	},
	{
		"empty successfully produces nothing",
		datastore.DeviceOpResults([]datastore.DeviceOpResult{}),
		[]proto.Message{},
		[]proto.Message{},
		true,
	},
	{
		"just timestamp, no lab config, no state config",
		[]datastore.DeviceOpResult{
			{
				Entity: &datastore.DeviceEntity{
					ID:      "476528fa-29f9-4bf8-8f39-92dc6c291ca1",
					Updated: time.Unix(42, 67),
				},
			},
		},
		[]proto.Message{
			&apibq.LabInventory{
				Id:          "476528fa-29f9-4bf8-8f39-92dc6c291ca1",
				UpdatedTime: timestampOrPanic(time.Unix(42, 67)),
				Device:      &lab.ChromeOSDevice{},
			},
		},
		[]proto.Message{
			&apibq.StateConfigInventory{
				Id:          "476528fa-29f9-4bf8-8f39-92dc6c291ca1",
				UpdatedTime: timestampOrPanic(time.Unix(42, 67)),
				State:       &lab.DutState{},
			},
		},
		true,
	},
}

func TestDeviceToBQMsgsSeq(t *testing.T) {
	t.Parallel()
	for i, tt := range testDeviceToBQMsgsSeqData {
		fmt.Printf("test case %d\n", i)
		t.Run(tt.name, func(t *testing.T) {
			out, stateOut, err := DeviceToBQMsgsSeq(tt.in)
			isGood := len(err.(errors.MultiError)) == 0
			fmt.Println(err)
			if isGood != tt.isGood {
				t.Errorf("error mismatch: expected (%v) got (%v) err was (%#v)", tt.isGood, isGood, err)
			}
			if diff := cmp.Diff(tt.out, out, protocmp.Transform()); diff != "" {
				t.Errorf("unexpected diff (%s)", diff)
			}
			if stateDiff := cmp.Diff(tt.stateOut, stateOut, protocmp.Transform()); stateDiff != "" {
				t.Errorf("unexpected diff (%s)", stateDiff)
			}
		})
	}
}

func timestampOrPanic(t time.Time) *timestamp.Timestamp {
	out, err := ptypes.TimestampProto(t)
	if err != nil {
		panic(err)
	}
	return out
}

func marshalOrPanic(item proto.Message) []byte {
	out, err := proto.Marshal(item)
	if err == nil {
		return out
	}
	panic(err)
}
