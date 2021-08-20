// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"
	"fmt"
	"strings"
	"time"

	ufsds "infra/unifiedfleet/app/model/datastore"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/config/go/api"
	"go.chromium.org/chromiumos/config/go/payload"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ConfigBundleKind is the datastore entity kind ConfigBundle.
const ConfigBundleKind string = "ConfigBundle"

// ConfigBundleEntity is a datastore entity that tracks a ConfigBundle.
type ConfigBundleEntity struct {
	_kind      string `gae:"$kind,ConfigBundle"`
	ID         string `gae:"$id"`
	ConfigData []byte `gae:",noindex"`
	Updated    time.Time
}

// GetProto returns the unmarshaled ConfigBundle.
func (e *ConfigBundleEntity) GetProto() (proto.Message, error) {
	p := &payload.ConfigBundle{}
	if err := proto.Unmarshal(e.ConfigData, p); err != nil {
		return nil, err
	}
	return p, nil
}

func GenerateCBEntityId(cb *payload.ConfigBundle) (string, error) {
	if len(cb.GetDesignList()) == 0 {
		return "", errors.Reason("Empty ConfigBundle DesignList").Err()
	}
	program := cb.GetDesignList()[0].GetProgramId().GetValue()
	design := cb.GetDesignList()[0].GetId().GetValue()

	if program == "" {
		return "", errors.Reason("Empty ConfigBundle ProgramId").Err()
	}
	if design == "" {
		return "", errors.Reason("Empty ConfigBundle DesignId").Err()
	}

	return fmt.Sprintf("%s-%s", program, design), nil
}

func newConfigBundleEntity(ctx context.Context, pm proto.Message) (cbEntity ufsds.FleetEntity, err error) {
	p, ok := pm.(*payload.ConfigBundle)
	if !ok {
		return nil, errors.Reason("Failed to create ConfigBundleEntity: %s", pm).Err()
	}

	id, err := GenerateCBEntityId(p)
	if err != nil {
		logging.Errorf(ctx, "Failed to generate ConfigBundle entity id: %s", err)
		return nil, err
	}

	configData, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "failed to marshal ConfigBundle %s", p).Err()
	}

	return &ConfigBundleEntity{
		ID:         id,
		ConfigData: configData,
		Updated:    time.Now().UTC(),
	}, nil
}

// UpdateConfigBundle updates ConfigBundle in datastore.
func UpdateConfigBundle(ctx context.Context, cb *payload.ConfigBundle) (*payload.ConfigBundle, error) {
	pm, err := ufsds.PutSingle(ctx, cb, newConfigBundleEntity)
	if err != nil {
		return nil, err
	}
	return pm.(*payload.ConfigBundle), nil
}

// GetConfigBundle returns ConfigBundle for the given id
// (${programId}-${designId}) from datastore.
func GetConfigBundle(ctx context.Context, id string) (rsp *payload.ConfigBundle, err error) {
	ids, err := extractIds(ctx, id)
	if err != nil {
		return nil, err
	}

	cb := &payload.ConfigBundle{
		DesignList: []*api.Design{
			{
				Id: &api.DesignId{
					Value: ids[1],
				},
				ProgramId: &api.ProgramId{
					Value: ids[0],
				},
			},
		},
	}
	pm, err := ufsds.Get(ctx, cb, newConfigBundleEntity)
	if err != nil {
		return nil, err
	}

	p, ok := pm.(*payload.ConfigBundle)
	if !ok {
		return nil, errors.Reason("Failed to create ConfigBundleEntity: %s", pm).Err()
	}
	return p, nil
}

// FlatConfigKind is the datastore entity kind FlatConfig.
const FlatConfigKind string = "FlatConfig"

// FlatConfigEntity is a datastore entity that tracks a FlatConfig.
type FlatConfigEntity struct {
	_kind      string `gae:"$kind,FlatConfig"`
	ID         string `gae:"$id"`
	ConfigData []byte `gae:",noindex"`
	Updated    time.Time
}

// GetProto returns the unmarshaled FlatConfig.
func (e *FlatConfigEntity) GetProto() (proto.Message, error) {
	p := &payload.FlatConfig{}
	if err := proto.Unmarshal(e.ConfigData, p); err != nil {
		return nil, err
	}
	return p, nil
}

func GenerateFCEntityId(fc *payload.FlatConfig) (string, error) {
	program := fc.GetHwDesign().GetProgramId().GetValue()
	design := fc.GetHwDesign().GetId().GetValue()

	if program == "" {
		return "", errors.Reason("Empty FlatConfig ProgramId").Err()
	}
	if design == "" {
		return "", errors.Reason("Empty FlatConfig DesignId").Err()
	}

	return fmt.Sprintf("%s-%s", program, design), nil
}

func newFlatConfigEntity(ctx context.Context, pm proto.Message) (fcEntity ufsds.FleetEntity, err error) {
	p, ok := pm.(*payload.FlatConfig)
	if !ok {
		return nil, errors.Reason("Failed to create FlatConfigEntity: %s", pm).Err()
	}

	id, err := GenerateFCEntityId(p)
	if err != nil {
		logging.Errorf(ctx, "Failed to generate FlatConfig entity id: %s", err)
		return nil, err
	}

	configData, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "failed to marshal FlatConfig %s", p).Err()
	}

	return &FlatConfigEntity{
		ID:         id,
		ConfigData: configData,
		Updated:    time.Now().UTC(),
	}, nil
}

// UpdateFlatConfig updates FlatConfig in datastore.
func UpdateFlatConfig(ctx context.Context, fc *payload.FlatConfig) (*payload.FlatConfig, error) {
	pm, err := ufsds.PutSingle(ctx, fc, newFlatConfigEntity)
	if err != nil {
		return nil, err
	}
	return pm.(*payload.FlatConfig), nil
}

// GetFlatConfig returns FlatConfig for the given id
// (${programId}-${designId}) from datastore.
func GetFlatConfig(ctx context.Context, id string) (rsp *payload.FlatConfig, err error) {
	ids, err := extractIds(ctx, id)
	if err != nil {
		return nil, err
	}

	fc := &payload.FlatConfig{
		HwDesign: &api.Design{
			Id: &api.DesignId{
				Value: ids[1],
			},
			ProgramId: &api.ProgramId{
				Value: ids[0],
			},
		},
	}
	pm, err := ufsds.Get(ctx, fc, newFlatConfigEntity)
	if err != nil {
		return nil, err
	}

	p, ok := pm.(*payload.FlatConfig)
	if !ok {
		return nil, errors.Reason("Failed to create FlatConfigEntity: %s", pm).Err()
	}
	return p, nil
}

func extractIds(ctx context.Context, id string) ([]string, error) {
	ids := strings.Split(id, "-")
	if len(ids) != 2 {
		logging.Errorf(ctx, "Faulty id value; please make sure the format is ${programId}-${designId}")
		return nil, status.Errorf(codes.InvalidArgument, ufsds.InvalidArgument)
	}
	return ids, nil
}
