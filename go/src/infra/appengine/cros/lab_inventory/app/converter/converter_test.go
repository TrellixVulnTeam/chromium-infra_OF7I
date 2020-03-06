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

	"go.chromium.org/chromiumos/infra/proto/go/lab"

	apibq "infra/appengine/cros/lab_inventory/api/bigquery"
	"infra/libs/cros/lab_inventory/datastore"
)

var testToBQLabInventoryData = []struct {
	name   string
	in     *datastore.DeviceOpResult
	out    *apibq.LabInventory
	isGood bool
}{
	{
		"empty",
		nil,
		nil,
		false,
	},
	{
		"just timestamp",
		&datastore.DeviceOpResult{
			Entity: &datastore.DeviceEntity{
				LabConfig: []byte{},
				Updated:   time.Unix(42, 67),
			},
		},
		&apibq.LabInventory{
			UpdatedTime: timestampOrPanic(time.Unix(42, 67)),
			Device:      &lab.ChromeOSDevice{},
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
				DutState: nil,
				Updated:  time.Unix(42, 67),
				Parent:   nil,
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
		true,
	},
}

func TestToBQLabInventory(t *testing.T) {
	t.Parallel()
	for _, tt := range testToBQLabInventoryData {
		t.Run(tt.name, func(t *testing.T) {
			out, err := ToBQLabInventory(tt.in)
			isGood := err == nil
			if isGood != tt.isGood {
				t.Errorf("error mismatch: expected (%v) got (%v) err was (%#v)", tt.isGood, isGood, err)
			}
			diff := cmp.Diff(tt.out, out)
			if diff != "" {
				msg := fmt.Sprintf("unexpected diff (%s)", diff)
				t.Errorf("%s", msg)
			}
		})
	}
}

var testToBQLabInventorySeqData = []struct {
	name   string
	in     datastore.DeviceOpResults
	out    []*apibq.LabInventory
	isGood bool
}{
	{
		"nil is failure",
		datastore.DeviceOpResults(nil),
		nil,
		false,
	},
	{
		"empty successfully produces nothing",
		datastore.DeviceOpResults([]datastore.DeviceOpResult{}),
		[]*apibq.LabInventory{},
		true,
	},
	{
		"just timestamp",
		[]datastore.DeviceOpResult{
			{
				Entity: &datastore.DeviceEntity{
					LabConfig: []byte{},
					Updated:   time.Unix(42, 67),
				},
			},
		},
		[]*apibq.LabInventory{
			{
				UpdatedTime: timestampOrPanic(time.Unix(42, 67)),
				Device:      &lab.ChromeOSDevice{},
			},
		},
		true,
	},
}

func TestToBQLabInventorySeq(t *testing.T) {
	t.Parallel()
	for _, tt := range testToBQLabInventorySeqData {
		t.Run(tt.name, func(t *testing.T) {
			out, err := ToBQLabInventorySeq(tt.in)
			isGood := err == nil
			if isGood != tt.isGood {
				t.Errorf("error mismatch: expected (%v) got (%v) err was (%#v)", tt.isGood, isGood, err)
			}
			diff := cmp.Diff(tt.out, out)
			if diff != "" {
				msg := fmt.Sprintf("unexpected diff (%s)", diff)
				t.Errorf("%s", msg)
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
