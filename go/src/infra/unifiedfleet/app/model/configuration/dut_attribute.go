// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"
	"time"

	ufsds "infra/unifiedfleet/app/model/datastore"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/luci/common/errors"
)

// DutAttributeKind is the datastore entity kind DutAttribute.
const DutAttributeKind string = "DutAttribute"

// DutAttributeEntity is a datastore entity that tracks a DutAttribute.
type DutAttributeEntity struct {
	_kind         string `gae:"$kind,DutAttribute"`
	ID            string `gae:"$id"`
	AttributeData []byte `gae:",noindex"`
	Updated       time.Time
}

// GetProto returns the unmarshaled DutAttribute.
func (e *DutAttributeEntity) GetProto() (proto.Message, error) {
	p := &api.DutAttribute{}
	if err := proto.Unmarshal(e.AttributeData, p); err != nil {
		return nil, err
	}
	return p, nil
}

func newDutAttributeEntity(ctx context.Context, pm proto.Message) (attrEntity ufsds.FleetEntity, err error) {
	p, ok := pm.(*api.DutAttribute)
	if !ok {
		return nil, errors.Reason("Failed to create DutAttributeEntity: %s", pm).Err()
	}

	attrData, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "failed to marshal DutAttribute %s", p).Err()
	}

	id := p.GetId().GetValue()
	if id == "" {
		return nil, errors.Reason("Empty DutAttribute ID").Err()
	}

	return &DutAttributeEntity{
		ID:            id,
		AttributeData: attrData,
		Updated:       time.Now().UTC(),
	}, nil
}

// UpdateDutAttribute updates DutAttribute in datastore.
func UpdateDutAttribute(ctx context.Context, attr *api.DutAttribute) (*api.DutAttribute, error) {
	pm, err := ufsds.PutSingle(ctx, attr, newDutAttributeEntity)
	if err != nil {
		return nil, err
	}
	return pm.(*api.DutAttribute), nil
}

// GetDutAttribute returns DutAttribute for the given id from datastore.
func GetDutAttribute(ctx context.Context, id string) (rsp *api.DutAttribute, err error) {
	attr := &api.DutAttribute{
		Id: &api.DutAttribute_Id{
			Value: id,
		},
	}
	pm, err := ufsds.Get(ctx, attr, newDutAttributeEntity)
	if err != nil {
		return nil, err
	}

	p, ok := pm.(*api.DutAttribute)
	if !ok {
		return nil, errors.Reason("Failed to create DutAttributeEntity: %s", pm).Err()
	}
	return p, nil
}
