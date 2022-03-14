// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components/servo"
	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

func servoUSBHasCROSStableImageExec(ctx context.Context, info *execs.ExecInfo) error {
	sv, err := info.Versioner().Cros(ctx, info.RunArgs.DUT.Name)
	if err != nil {
		return errors.Annotate(err, "servo usb-key has cros stable image").Err()
	}
	expectedImage := sv.OSImage
	if expectedImage == "" {
		return errors.Reason("servo usb-key has cros stable image: stable image is not specified").Err()
	}
	run := info.NewRunner(info.RunArgs.DUT.ServoHost.Name)
	servod := info.NewServod()
	logger := info.NewLogger()
	usbPath, err := servo.USBDrivePath(ctx, false, run, servod, logger)
	if err != nil {
		return errors.Annotate(err, "servo usb-key has cros stable image").Err()
	}
	imageName, err := servo.ChromeOSImageNameFromUSBDrive(ctx, usbPath, run, servod, logger)
	if err != nil {
		return errors.Annotate(err, "servo usb-key has cros stable image").Err()
	}
	if strings.Contains(expectedImage, imageName) {
		log.Info(ctx, "The image %q found on USB-key and match to stable version", imageName)
		return nil
	}
	return errors.Reason("servo usb-key has cros stable image: expected %q but found %q", expectedImage, imageName).Err()
}

func init() {
	execs.Register("servo_usbkey_has_stable_image", servoUSBHasCROSStableImageExec)
}
