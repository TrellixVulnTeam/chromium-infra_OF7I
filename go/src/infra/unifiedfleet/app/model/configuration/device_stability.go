// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/config/go/test/dut"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DeviceStabilityKind is the datastore entity kind for device stability.
const DeviceStabilityKind string = "DeviceStability"

// DeviceStabilityEntity is a datastore entity that tracks a device stability record .
type DeviceStabilityEntity struct {
	_kind         string `gae:"$kind,DeviceStability"`
	ID            string `gae:"$id"`
	StabilityData []byte `gae:",noindex"`
	Updated       time.Time
}

// GetProto returns the unmarshaled DeviceStability.
func (e *DeviceStabilityEntity) GetProto() (proto.Message, error) {
	p := &dut.DeviceStability{}
	if err := proto.Unmarshal(e.StabilityData, p); err != nil {
		return nil, err
	}
	return p, nil
}

// GetDeviceStability returns deviceStability for the given id from datastore.
func GetDeviceStability(ctx context.Context, id string) (*dut.DeviceStability, error) {
	e := &DeviceStabilityEntity{ID: id}
	err := datastore.Get(ctx, e)
	if err == nil {
		p, err := e.GetProto()
		if err != nil {
			return nil, err
		}
		return p.(*dut.DeviceStability), nil
	}
	if datastore.IsErrNoSuchEntity(err) {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Entity not found %s", id))
	}
	return nil, status.Errorf(codes.Unknown, fmt.Sprintf("fail to get entity for %s: %s", id, err))
}

func UpdateDeviceStability(ctx context.Context, id string, ds *dut.DeviceStability) error {
	data, err := proto.Marshal(ds)
	if err != nil {
		return errors.Annotate(err, "failed to marshal DeviceStability").Err()
	}

	dsEntity := DeviceStabilityEntity{ID: id, StabilityData: data, Updated: time.Now().UTC()}
	if err := datastore.Put(ctx, &dsEntity); err != nil {
		return errors.Annotate(err, "failed to save device stability into datastore").Err()
	}
	return nil
}
