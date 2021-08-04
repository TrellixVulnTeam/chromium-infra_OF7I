// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/log"
)

// hasCrosImageStableVersionActionExec verifies that DUT provides ChromeOS image name as part of stable version.
// Example: board-release/R90-13816.47.0.
func hasCrosImageStableVersionActionExec(ctx context.Context, args *RunArgs) error {
	if args.DUT != nil && args.DUT.StableVersion != nil {
		image := args.DUT.StableVersion.CrosImage
		log.Debug(ctx, "Stable version for cros: %q", image)
		if image != "" && strings.Contains(image, "/") {
			return nil
		}
	}
	return errors.Reason("stable version does not have cros image name").Err()
}

// hasFwVersionStableVersionActionExec verifies that DUT provides ChromeOS firmware version name as part of stable version.
// Example: Google_Board.13434.261.0.
func hasFwVersionStableVersionActionExec(ctx context.Context, args *RunArgs) error {
	if args.DUT != nil && args.DUT.StableVersion != nil {
		version := args.DUT.StableVersion.CrosFirmwareVersion
		log.Debug(ctx, "Stable version for firmware version: %q", version)
		if version != "" {
			return nil
		}
	}
	return errors.Reason("stable version does not have firmware version").Err()
}

// hasFwImageStableVersionActionExec verifies that DUT provides ChromeOS firmware image name as part of stable version.
// Example: board-firmware/R87-13434.261.0
func hasFwImageStableVersionActionExec(ctx context.Context, args *RunArgs) error {
	if args.DUT != nil && args.DUT.StableVersion != nil {
		image := args.DUT.StableVersion.CrosFirmwareImage
		log.Debug(ctx, "Stable version for firmware image: %q", image)
		if image != "" && strings.Contains(image, "/") {
			return nil
		}
	}
	return errors.Reason("stable version does not have firmware version").Err()
}

func init() {
	execMap["has_stable_version_cros_image"] = hasCrosImageStableVersionActionExec
	execMap["has_stable_version_fw_version"] = hasFwVersionStableVersionActionExec
	execMap["has_stable_version_fw_image"] = hasFwImageStableVersionActionExec
}
