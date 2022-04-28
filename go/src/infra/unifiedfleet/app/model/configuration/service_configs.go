// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"
	"fmt"
	"time"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufsds "infra/unifiedfleet/app/model/datastore"
)

const (
	serviceConfigID = "UFS"
	lowerHex        = "000000"
)

// ServiceConfig we stored for ufs.
type ServiceConfig struct {
	// The last checked vm mac address. Used for vm mac address auto generation.
	LastCheckedVMMacAddress string
}

// ServiceConfigEntity is a datastore entity that records service-level configs.
type ServiceConfigEntity struct {
	// ServiceConfig is the datastore entity kind for service-level configs.
	_kind      string        `gae:"$kind,ServiceConfig"`
	ID         string        `gae:"$id"`
	Data       ServiceConfig `gae:",noindex"`
	UpdateTime time.Time
}

func GetServiceConfig(ctx context.Context) (*ServiceConfig, error) {
	e := ServiceConfigEntity{ID: serviceConfigID}
	err := datastore.Get(ctx, &e)
	if err == nil {
		return &e.Data, nil
	}
	if datastore.IsErrNoSuchEntity(err) {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Entity not found %s", serviceConfigID))
	}
	return nil, status.Errorf(codes.Internal, ufsds.InternalError)
}

func UpdateServiceConfig(ctx context.Context, sc *ServiceConfig) error {
	scEntity := ServiceConfigEntity{ID: serviceConfigID, Data: *sc}
	scEntity.UpdateTime = time.Now().UTC()
	if err := datastore.Put(ctx, &scEntity); err != nil {
		logging.Errorf(ctx, "Failed to update service config: %s", err)
		fmt.Println(err)
		return status.Errorf(codes.Internal, ufsds.InternalError)
	}
	return nil
}

func InitEmptyServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		LastCheckedVMMacAddress: lowerHex,
	}
}
