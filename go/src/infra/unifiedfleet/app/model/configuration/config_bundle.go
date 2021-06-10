// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	ufsds "infra/unifiedfleet/app/model/datastore"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/config/go/payload"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// ConfigBundleKind is the datastore entity kind ConfigBundle.
const ConfigBundleKind string = "ConfigBundle"

// ConfigBundleEntity is a datastore entity that tracks a ConfigBundle.
type ConfigBundleEntity struct {
	_kind      string    `gae:"$kind,ConfigBundle"`
	ID         string    `gae:"$id"`
	ConfigData []byte    `gae:",noindex"`
	Updated    time.Time `gae:"updated_time"`
}

// GetProto returns the unmarshaled ConfigBundle.
func (e *ConfigBundleEntity) GetProto() (proto.Message, error) {
	p := &payload.ConfigBundle{}
	if err := proto.Unmarshal(e.ConfigData, p); err != nil {
		return nil, err
	}
	return p, nil
}

func generateCBEntityId(cb *payload.ConfigBundle) (string, error) {
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
	defer func() {
		if r := recover(); r != nil {
			logging.Errorf(ctx, "Failed to create ConfigBundleEntity: %s", r)
			debug.PrintStack()
			err = errors.Reason("Failed to create ConfigBundleEntity: %s", r).Err()
		}
	}()
	p := pm.(*payload.ConfigBundle)

	id, err := generateCBEntityId(p)
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
