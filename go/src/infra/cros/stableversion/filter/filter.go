// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package filter

import (
	labPlatform "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
)

// WithModel takes a stable version config description and a model and returns a
// fresh StableVersions data object with that only has entries corresponding to the
// given model.
// sv cannot be nil. if model is "", then return everything.
func WithModel(sv *labPlatform.StableVersions, targetModel string) *labPlatform.StableVersions {
	var cros []*labPlatform.StableCrosVersion
	var faft []*labPlatform.StableFaftVersion
	var firmware []*labPlatform.StableFirmwareVersion
	if targetModel == "" {
		return sv
	}
	// In order to get the CrOS version for the model in question,
	// we check the firmware version before the CrOS version and use
	// that entry to infer the correct board.
	inferredBoard := ""
	for _, item := range sv.GetFaft() {
		if item.GetKey().GetModelId().GetValue() == targetModel {
			faft = append(faft, item)
		}
	}
	for _, item := range sv.GetFirmware() {
		if item.GetKey().GetModelId().GetValue() == targetModel {
			board := item.GetKey().GetBuildTarget().GetName()
			inferredBoard = board
			firmware = append(firmware, item)
		}
	}
	for _, item := range sv.GetCros() {
		if item.GetKey().GetBuildTarget().GetName() == inferredBoard {
			cros = append(cros, item)
		}
	}
	return &labPlatform.StableVersions{
		Cros:     cros,
		Faft:     faft,
		Firmware: firmware,
	}
}
