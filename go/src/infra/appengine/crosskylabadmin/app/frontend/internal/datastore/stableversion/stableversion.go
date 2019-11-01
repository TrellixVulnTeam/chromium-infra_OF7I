// Copyright 2019 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package stableversion provides functions to store stableversion info in datastore
package stableversion

import (
	"context"
	"fmt"
	"strings"

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
)

const (
	crosStableVersionKind     = "crosStableVersion"
	faftStableversionKind     = "faftStableVersion"
	firmwareStableVersionKind = "firmwareStableVersion"
)

type crosStableVersionEntity struct {
	_kind string `gae:"$kind,crosStableVersion"`
	ID    string `gae:"$id"`
	Cros  string
}

type faftStableVersionEntity struct {
	_kind string `gae:"$kind,faftStableVersion"`
	ID    string `gae:"$id"`
	Faft  string
}

type firmwareStableVersionEntity struct {
	_kind    string `gae:"$kind,firmwareStableVersion"`
	ID       string `gae:"$id"`
	Firmware string
}

const separator = ";"

// GetCrosStableVersion gets a stable version for ChromeOS from datastore
func GetCrosStableVersion(ctx context.Context, buildTarget string) (string, error) {
	if buildTarget == "" {
		return "", fmt.Errorf("GetCrosStableVersion: buildTarget cannot be empty")
	}
	entity := &crosStableVersionEntity{ID: buildTarget}
	if err := datastore.Get(ctx, entity); err != nil {
		return "", errors.Annotate(err, "GetCrosStableVersion").Err()
	}
	return entity.Cros, nil
}

// PutCrosStableVersion writes a stable version for ChromeOS to datastore
func PutCrosStableVersion(ctx context.Context, buildTarget string, cros string) error {
	if buildTarget == "" {
		return fmt.Errorf("PutCrosStableVersion: buildTarget cannot be nil")
	}
	if cros == "" {
		return fmt.Errorf("PutCrosStableVersion: cros cannot be nil")
	}
	entity := &crosStableVersionEntity{ID: buildTarget, Cros: cros}
	if err := datastore.Put(ctx, entity); err != nil {
		return errors.Annotate(err, "PutCrosStableVersion").Err()
	}
	return nil
}

// GetFirmwareStableVersion takes a buildtarget and a model and produces a firmware stable version from datastore
func GetFirmwareStableVersion(ctx context.Context, buildTarget string, model string) (string, error) {
	key, err := combineBuildTargetModel(buildTarget, model)
	if err != nil {
		return "", errors.Annotate(err, "GetFirmwareStableVersion").Err()
	}
	entity := &firmwareStableVersionEntity{ID: key}
	if err := datastore.Get(ctx, entity); err != nil {
		return "", errors.Annotate(err, "GetFirmwareStableVersion").Err()
	}
	return entity.Firmware, nil
}

// PutFirmwareStableVersion takes a buildtarget, a model, and a firmware stableversion and persists it to datastore
func PutFirmwareStableVersion(ctx context.Context, buildTarget string, model string, firmware string) error {
	key, err := combineBuildTargetModel(buildTarget, model)
	if err != nil {
		return err
	}
	if firmware == "" {
		return fmt.Errorf("PutFirmwareStableVersion: firmware cannot be nil")
	}
	entity := &firmwareStableVersionEntity{ID: key, Firmware: firmware}
	if err := datastore.Put(ctx, entity); err != nil {
		return errors.Annotate(err, "PutFirmwareStableVersion").Err()
	}
	return nil
}

// GetFaftStableVersion takes a model and a buildtarget and produces a faft stable version from datastore
func GetFaftStableVersion(ctx context.Context, buildTarget string, model string) (string, error) {
	key, err := combineBuildTargetModel(buildTarget, model)
	if err != nil {
		return "", errors.Annotate(err, "GetFaftStableVersion").Err()
	}
	entity := &faftStableVersionEntity{ID: key}
	if err := datastore.Get(ctx, entity); err != nil {
		return "", errors.Annotate(err, "GetFaftStableVersion").Err()
	}
	return entity.Faft, nil
}

// PutFaftStableVersion takes a model, buildtarget, and faft stableversion and persists it to datastore
func PutFaftStableVersion(ctx context.Context, buildTarget string, model string, faft string) error {
	key, err := combineBuildTargetModel(buildTarget, model)
	if err != nil {
		return err
	}
	if faft == "" {
		return fmt.Errorf("PutFaftStableVersion: faft cannot be nil")
	}
	entity := &faftStableVersionEntity{ID: key, Faft: faft}
	if err := datastore.Put(ctx, entity); err != nil {
		return errors.Annotate(err, "PutFaftStableVersion").Err()
	}
	return nil
}

func combineBuildTargetModel(buildTarget string, model string) (string, error) {
	if model == "" {
		return "", fmt.Errorf("combineBuildTargetModel: model cannot be empty")
	}
	if buildTarget == "" {
		return "", fmt.Errorf("combineBuildTargetModel: buildTarget cannot be empty")
	}
	if strings.Contains(model, separator) {
		return "", fmt.Errorf("combineBuildTargetModel: model cannot contain `%s`", separator)
	}
	if strings.Contains(buildTarget, separator) {
		return "", fmt.Errorf("combineBuildTargetModel: buildTarget cannot contain `%s`", separator)
	}
	return fmt.Sprintf("%s%s%s", buildTarget, separator, model), nil
}
