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

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	libsv "infra/libs/cros/stableversion"
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

// GetCrosStableVersion gets a stable version for ChromeOS from datastore
func GetCrosStableVersion(ctx context.Context, buildTarget string) (string, error) {
	if buildTarget == "" {
		return "", fmt.Errorf("GetCrosStableVersion: buildTarget cannot be empty")
	}
	entity := &crosStableVersionEntity{ID: libsv.BuildTargetKey(buildTarget)}
	if err := datastore.Get(ctx, entity); err != nil {
		return "", errors.Annotate(err, "GetCrosStableVersion").Err()
	}
	return entity.Cros, nil
}

// PutSingleCrosStableVersion is a convenience wrapper around PutManyCrosStableVersion
func PutSingleCrosStableVersion(ctx context.Context, buildTarget string, cros string) error {
	return PutManyCrosStableVersion(ctx, map[string]string{buildTarget: cros})
}

// PutManyCrosStableVersion writes many stable versions for ChromeOS to datastore
func PutManyCrosStableVersion(ctx context.Context, crosOfBuildTarget map[string]string) error {
	removeEmptyKeyOrValue(ctx, crosOfBuildTarget)
	var entities []*crosStableVersionEntity
	for buildTarget, cros := range crosOfBuildTarget {
		entities = append(entities, &crosStableVersionEntity{ID: libsv.BuildTargetKey(buildTarget), Cros: cros})
	}
	if err := datastore.Put(ctx, entities); err != nil {
		return errors.Annotate(err, "PutManyCrosStableVersion").Err()
	}
	return nil
}

// GetFirmwareStableVersion takes a buildtarget and a model and produces a firmware stable version from datastore
func GetFirmwareStableVersion(ctx context.Context, buildTarget string, model string) (string, error) {
	key, err := libsv.JoinBuildTargetModel(buildTarget, model)
	if err != nil {
		return "", errors.Annotate(err, "GetFirmwareStableVersion").Err()
	}
	entity := &firmwareStableVersionEntity{ID: key}
	if err := datastore.Get(ctx, entity); err != nil {
		return "", errors.Annotate(err, "GetFirmwareStableVersion").Err()
	}
	return entity.Firmware, nil
}

// PutSingleFirmwareStableVersion is a convenience wrapper around PutManyFirmwareStableVersion
func PutSingleFirmwareStableVersion(ctx context.Context, buildTarget string, model string, firmware string) error {
	key, err := libsv.JoinBuildTargetModel(buildTarget, model)
	if err != nil {
		return err
	}
	return PutManyFirmwareStableVersion(ctx, map[string]string{key: firmware})
}

// PutManyFirmwareStableVersion takes a map from build_target+model keys to firmware versions and persists it to datastore
func PutManyFirmwareStableVersion(ctx context.Context, firmwareOfJoinedKey map[string]string) error {
	removeEmptyKeyOrValue(ctx, firmwareOfJoinedKey)
	var entities []*firmwareStableVersionEntity
	for key, firmware := range firmwareOfJoinedKey {
		entities = append(entities, &firmwareStableVersionEntity{ID: key, Firmware: firmware})
	}
	if err := datastore.Put(ctx, entities); err != nil {
		return errors.Annotate(err, "PutManyFirmwareStableVersion").Err()
	}
	return nil
}

// GetFaftStableVersion takes a model and a buildtarget and produces a faft stable version from datastore
func GetFaftStableVersion(ctx context.Context, buildTarget string, model string) (string, error) {
	key, err := libsv.JoinBuildTargetModel(buildTarget, model)
	if err != nil {
		return "", errors.Annotate(err, "GetFaftStableVersion").Err()
	}
	entity := &faftStableVersionEntity{ID: key}
	if err := datastore.Get(ctx, entity); err != nil {
		return "", errors.Annotate(err, "GetFaftStableVersion").Err()
	}
	return entity.Faft, nil
}

// PutSingleFaftStableVersion is a convenience wrapper around PutManyFaftStableVersion
func PutSingleFaftStableVersion(ctx context.Context, buildTarget string, model string, faft string) error {
	key, err := libsv.JoinBuildTargetModel(buildTarget, model)
	if err != nil {
		return err
	}
	return PutManyFaftStableVersion(ctx, map[string]string{key: faft})
}

// PutManyFaftStableVersion takes a model, buildtarget, and faft stableversion and persists it to datastore
func PutManyFaftStableVersion(ctx context.Context, faftOfJoinedKey map[string]string) error {
	removeEmptyKeyOrValue(ctx, faftOfJoinedKey)
	var entities []*faftStableVersionEntity
	for key, faft := range faftOfJoinedKey {
		entities = append(entities, &faftStableVersionEntity{ID: key, Faft: faft})
	}
	if err := datastore.Put(ctx, entities); err != nil {
		return errors.Annotate(err, "PutManyFaftStableVersion").Err()
	}
	return nil
}

// removeEmptyKeyOrValue destructively drops empty keys or values from versionMap
func removeEmptyKeyOrValue(ctx context.Context, versionMap map[string]string) {
	removedTally := 0
	for k, v := range versionMap {
		if k == "" || v == "" {
			logging.Infof(ctx, "removed non-conforming key-value pair (%s) -> (%s)", k, v)
			delete(versionMap, k)
			removedTally++
			continue
		}
	}
	if removedTally > 0 {
		logging.Infof(ctx, "removed (%d) pairs for containing \"\" as key or value", removedTally)
	}
}
