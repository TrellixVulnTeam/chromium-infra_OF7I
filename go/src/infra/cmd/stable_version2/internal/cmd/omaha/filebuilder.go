// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package omaha

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"

	"infra/cmd/stable_version2/internal/utils"

	sv "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	"go.chromium.org/luci/common/logging"
	svlib "infra/libs/cros/stableversion"
)

// 1. Determine the map from board+model to CrOS version
//    If a board is present with no model in the old file,
//    then it is a legacy entry and will be replaced with
//    a board+model stable version entries for all the relevant
//    models.
// 2. Determine all the models and all the versions that each board
//    supports.
// 3. Determine the map from the CrOS version to the firmware version
//    for every model.
// 4. Pick the best CrOS version for each model.
// 5. Look up the corresponding firmware version for each model,version pair

// FileBuilder takes the old file, the new CrOS entries from Omaha, a Google Storage client,
// and an output directory and returns the new stable version file.
func FileBuilder(
	ctx context.Context,
	oldSV *sv.StableVersions,
	newSV []*sv.StableCrosVersion,
	fvFunc FirmwareVersionFunc,
) (*sv.StableVersions, error) {
	boardVersionMaps := getBoardVersionMaps(ctx, oldSV, newSV)
	stableVersions, err := updateStableVersions(
		ctx,
		oldSV,
		boardVersionMaps,
		fvFunc,
	)
	if err != nil {
		return nil, err
	}
	svlib.SortSV(stableVersions)
	return stableVersions, nil
}

// getBoardVersionMaps returns a map from the board names to
// a version description.
func getBoardVersionMaps(
	ctx context.Context,
	oldSV *sv.StableVersions,
	newSV []*sv.StableCrosVersion,
) map[string]*boardVersionMap {
	out := make(map[string]*boardVersionMap)
	for _, old := range oldSV.Cros {
		bt := old.GetKey().GetBuildTarget().GetName()
		model := old.GetKey().GetModelId().GetValue()
		if _, ok := out[bt]; !ok {
			out[bt] = newBoardVersionMap()
		}

		if model == "" {
			out[bt].oldBoardVersion = old.GetVersion()
		} else {
			out[bt].oldModelMap[model] = old.GetVersion()
		}
	}
	for _, newItem := range newSV {
		bt := newItem.GetKey().GetBuildTarget().GetName()
		if bt == "" {
			logging.Debugf(ctx, "skipping new item %#v", newItem)
			continue
		}
		version := newItem.GetVersion()
		if version == "" {
			logging.Debugf(ctx, "buildTarget has blank version", version)
			continue
		}
		if _, ok := out[bt]; !ok {
			out[bt] = newBoardVersionMap()
		}
		out[bt].omahaVersion = version
	}
	return out
}

// getCrosFirmwareVersion takes a map from board names to relevant version info
// and determines what the corresponding CrOS and firmware versions should be
func getCrosFirmwareVersion(
	ctx context.Context,
	oldSV *sv.StableVersions,
	board string,
	versionMap *boardVersionMap,
	fvFunc FirmwareVersionFunc,
) ([]*sv.StableCrosVersion, []*sv.StableFirmwareVersion, error) {
	var crosArr []*sv.StableCrosVersion
	var fwArr []*sv.StableFirmwareVersion

	// primary key: crosv, secondary key: model name
	allFirmwareVersions := make(map[string]map[string]*sv.StableFirmwareVersion)
	// all Models reported across all versions of the cros board
	allModels := make(map[string]bool)

	for crosv := range versionMap.allCrosVersions() {
		res, err := fvFunc(
			ctx,
			board,
			crosv,
		)
		if err != nil {
			return nil, nil, err
		}

		if item := allFirmwareVersions[crosv]; item == nil {
			allFirmwareVersions[crosv] = make(map[string]*sv.StableFirmwareVersion)
		}

		for _, fw := range res {
			model := fw.GetKey().GetModelId().GetValue()
			allFirmwareVersions[crosv][model] = fw
			allModels[model] = true
		}
	}

	for model := range allModels {
		crosv, err := versionMap.bestVersion(model)
		if err != nil {
			return nil, nil, err
		}

		// allFirmwareVersions[crosv] will definitely exist
		// so we aren't looking up a value in a nil map
		firmwareEntry := allFirmwareVersions[crosv][model]
		crosEntry := utils.MakeSpecificCrOSSV(board, model, crosv)
		// explicitly fall back to the old firmware version.
		// This is a rare operation so it's okay if we have to scan
		// all the old firmware versions.
		if firmwareEntry == nil {
			logging.Infof(ctx, "Falling back to linear scan for model %q", model)
			for _, fw := range oldSV.Firmware {
				b := fw.GetKey().GetBuildTarget().GetName()
				m := fw.GetKey().GetModelId().GetValue()
				if b == board && m == model {
					firmwareEntry = fw
					// if we are falling back to the old version explicitly,
					// then it is safe to assume that the oldBoardVersion that
					// we were given is compatible with the entry in oldSV.
					crosEntry = utils.MakeSpecificCrOSSV(b, m, versionMap.oldBoardVersion)
					break
				}
			}
		}
		if firmwareEntry == nil {
			logging.Infof(ctx, "novel model %q in board %q", board, model)
		} else {
			crosArr = append(crosArr, crosEntry)
			fwArr = append(fwArr, firmwareEntry)
		}
	}
	return crosArr, fwArr, nil
}

// updateStableVersions produces an updated StableVersions file
// given an old file, the versions associated with each board,
// and a way of retrieving firmware versions.
func updateStableVersions(
	ctx context.Context,
	oldSV *sv.StableVersions,
	boardVersionMaps map[string]*boardVersionMap,
	fvFunc FirmwareVersionFunc,
) (*sv.StableVersions, error) {
	var cros []*sv.StableCrosVersion
	var firmware []*sv.StableFirmwareVersion
	for board, versionMap := range boardVersionMaps {
		if board == "" {
			continue
		}

		c, f, err := getCrosFirmwareVersion(
			ctx,
			oldSV,
			board,
			versionMap,
			fvFunc,
		)
		if err != nil {
			return nil, err
		}

		cros = append(cros, c...)
		firmware = append(firmware, f...)
	}
	newSV := proto.Clone(oldSV).(*sv.StableVersions)
	newSV.Cros = cros
	newSV.Firmware = firmware
	// newSV.Faft remains unchanged
	return newSV, nil
}

type boardVersionMap struct {
	// The version from Omaha is always tied to the board,
	// it will never be specific to a model.
	omahaVersion string
	// A local board version will be present in a "legacy"
	// stable version entry. New CrOS entries will be tied
	// to the model
	oldBoardVersion string

	oldModelMap map[string]string
}

// newBoardVersionMap returns an empty boardVersionMap
func newBoardVersionMap() *boardVersionMap {
	return &boardVersionMap{
		omahaVersion:    "",
		oldBoardVersion: "",
		oldModelMap:     make(map[string]string),
	}
}

// bestVersion determines the best available version for a given model
// between the config file and Omaha.
func (m *boardVersionMap) bestVersion(model string) (string, error) {
	newVersion := m.omahaVersion
	oldVersion := ""
	if v, ok := m.oldModelMap[model]; ok {
		oldVersion = v
	} else {
		oldVersion = m.oldBoardVersion
	}

	newVersionValid := versionOk(newVersion)
	oldVersionValid := versionOk(oldVersion)

	// Both versions are valid, pick the newer one.
	// The "old" version can be newer than the "new" version
	// if someone has explicitly overridden the stable version file
	// for a given model, which is expected during bring-up.
	if oldVersionValid && newVersionValid {
		if versionCmp(newVersion, oldVersion) > 0 {
			return newVersion, nil
		}
		return oldVersion, nil
	}

	if !oldVersionValid && newVersionValid {
		return newVersion, nil
	}

	if !newVersionValid && oldVersionValid {
		return oldVersion, nil
	}

	return "", fmt.Errorf("both versions are invalid new: %q and old: %q", newVersion, oldVersion)
}

// allCrosVersions gets all the CrOS versions associated with a given board.
// We may need to look up more than one version per board.
func (m *boardVersionMap) allCrosVersions() map[string]bool {
	out := make(map[string]bool)
	if m.omahaVersion != "" {
		out[m.omahaVersion] = true
	}
	if m.oldBoardVersion != "" {
		out[m.oldBoardVersion] = true
	}
	for _, v := range m.oldModelMap {
		out[v] = true
	}
	return out
}

// FirmwareVersionFunc is a type that takes a board and version
// and returns a firmware version
type FirmwareVersionFunc func(
	ctx context.Context,
	board string,
	version string,
) ([]*sv.StableFirmwareVersion, error)

// MakeFirmwareVersionFunc takes a gslibClient and a temporary directory
// and returns a function that can convert a board and version into
// stable firmware versions.
func MakeFirmwareVersionFunc(gsc gslibClient, outDir string) FirmwareVersionFunc {
	return func(
		ctx context.Context,
		board string,
		version string,
	) ([]*sv.StableFirmwareVersion, error) {
		var cros []*sv.StableCrosVersion
		cros = append(cros, utils.MakeCrOSSV(board, version))
		out, err := getGSFirmwareSV(ctx, gsc, outDir, cros)
		return out, err
	}
}
