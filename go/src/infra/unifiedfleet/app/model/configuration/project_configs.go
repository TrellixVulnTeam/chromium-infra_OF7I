// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ProjectConfigKind is the datastore entity kind for storing the project configs.
const ProjectConfigKind string = "ProjectConfig"

// ProjectConfigEntity is a datastore entity that stores the project configs.
type ProjectConfigEntity struct {
	_kind            string `gae:"$kind,ProjectConfig"`
	Name             string `gae:"$id"`
	DailyDumpTimeStr string
}

// SaveProjectConfig saves project config to database
func SaveProjectConfig(ctx context.Context, e *ProjectConfigEntity) error {
	if err := datastore.Put(ctx, e); err != nil {
		logging.Errorf(ctx, "Failed to put project config in datastore: %s", err)
		return status.Errorf(codes.Internal, err.Error())
	}
	return nil
}

// GetProjectConfig gets project config from database
func GetProjectConfig(ctx context.Context, name string) (*ProjectConfigEntity, error) {
	entity := &ProjectConfigEntity{
		Name: name,
	}
	if err := datastore.Get(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}
