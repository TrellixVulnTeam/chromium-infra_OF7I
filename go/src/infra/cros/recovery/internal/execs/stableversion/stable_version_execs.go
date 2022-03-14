// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package stableversion

import (
	"context"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

// hasCrosImageStableVersionActionExec verifies that DUT provides ChromeOS image name as part of stable version.
// Example: board-release/R90-13816.47.0.
func hasCrosImageStableVersionActionExec(ctx context.Context, info *execs.ExecInfo) error {
	if info.RunArgs.DUT != nil {
		sv, err := info.Versioner().Cros(ctx, info.RunArgs.DUT.Name)
		if err != nil {
			return errors.Annotate(err, "cros has stable version").Err()
		}
		log.Debug(ctx, "Stable version for cros: %q", sv.OSImage)
		if sv.OSImage != "" && strings.Contains(sv.OSImage, "/") {
			return nil
		}
	}
	return errors.Reason("cros has stable version: not found").Err()
}

// hasFwVersionStableVersionActionExec verifies that DUT provides ChromeOS firmware version name as part of stable version.
// Example: Google_Board.13434.261.0.
func hasFwVersionStableVersionActionExec(ctx context.Context, info *execs.ExecInfo) error {
	if info.RunArgs.DUT != nil {
		sv, err := info.Versioner().Cros(ctx, info.RunArgs.DUT.Name)
		if err != nil {
			return errors.Annotate(err, "cros has stable firmware version").Err()
		}
		log.Debug(ctx, "Stable version for firmware version: %q", sv.FwVersion)
		if sv.FwVersion != "" {
			return nil
		}
	}
	return errors.Reason("cros has stable firmware version: not found").Err()
}

// hasFwImageStableVersionActionExec verifies that DUT provides ChromeOS firmware image name as part of stable version.
// Example: board-firmware/R87-13434.261.0
func hasFwImageStableVersionActionExec(ctx context.Context, info *execs.ExecInfo) error {
	if info.RunArgs.DUT != nil {
		sv, err := info.Versioner().Cros(ctx, info.RunArgs.DUT.Name)
		if err != nil {
			return errors.Annotate(err, "cros has stable firmware image version").Err()
		}
		log.Debug(ctx, "Stable version for firmware image: %q", sv.FwImage)
		if sv.FwImage != "" && strings.Contains(sv.FwImage, "/") {
			return nil
		}
	}
	return errors.Reason("cros has stable firmware image version: not found").Err()
}

func init() {
	execs.Register("has_stable_version_cros_image", hasCrosImageStableVersionActionExec)
	execs.Register("has_stable_version_fw_version", hasFwVersionStableVersionActionExec)
	execs.Register("has_stable_version_fw_image", hasFwImageStableVersionActionExec)
}
