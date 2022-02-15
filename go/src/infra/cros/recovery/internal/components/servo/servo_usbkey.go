// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/internal/components/linux"
	"infra/cros/recovery/logger"
)

const (
	// Minimum time  to execute command on the hosts as 30 seconds.
	minRunTimeout = 30 * time.Second
)

// USBDrivePath read usb-path from servod and check readability of the USB per request.
func USBDrivePath(ctx context.Context, fileCheck bool, run components.Runner, servod components.Servod, log logger.Logger) (string, error) {
	v, err := servod.Get(ctx, "image_usbkey_dev")
	if err != nil {
		return "", errors.Annotate(err, "usb-drive path").Err()
	}
	usbPath := v.GetString_()
	if usbPath == "" {
		return "", errors.Reason("usb-drive path: usb-path is empty").Err()
	}
	if fileCheck {
		if out, err := run(ctx, time.Minute, "fdisk", "-l", usbPath); err != nil {
			return "", errors.Annotate(err, "usb-drive path: file check by fdisk").Err()
		} else {
			log.Debug("USB-key fdisk check results:\n%s", out)
		}
	}
	return usbPath, nil
}

const (
	// Path where USB-key will be mounted.
	usbMountPathGlob    = "/media/servo_usb/port_%d"
	releaseInfoFilename = "etc/lsb-release"
)

var (
	// Check if the image build is test image.
	crosTestImageTrack = regexp.MustCompile(`RELEASE_TRACK=.*test`)
	// Read image version and target-board from etc/lsb-release.
	crosTestImageName = regexp.MustCompile(`CHROMEOS_RELEASE_BUILDER_PATH=([\w\W]*)`)
)

// ChromeOSImageNameFromUSBDrive reads image name from USB-drive plugged to servo.
//
// The version will be read from partition 3 of the ChromeOS image.
func ChromeOSImageNameFromUSBDrive(ctx context.Context, usbPath string, run components.Runner, servod components.Servod, log logger.Logger) (string, error) {
	mountDst := fmt.Sprintf(usbMountPathGlob, servod.Port())
	unmount := func() {
		if err := linux.UnmountDrive(ctx, run, mountDst); err != nil {
			log.Debug("ChromeOS image name from USB drive (not critical): %s", err)
		}
	}
	// Unmount if there is an existing stale mount.
	unmount()
	// Set defer to unmount the device in any case to left lace clean.
	defer unmount()
	// ChromeOS root fs is in /dev/sdx3
	// The version is present in partition 3 of ChromeOS image.
	mountSrc := usbPath + "3"
	if err := linux.MountDrive(ctx, run, mountDst, mountSrc); err != nil {
		return "", errors.Annotate(err, "cros image name from usb drive").Err()
	}

	// We using only test image in the lab so to be sure we need verify that image is test image
	releaseInfoPath := fmt.Sprintf("%s/%s", mountDst, releaseInfoFilename)
	out, err := run(ctx, time.Minute, "cat", releaseInfoPath)
	if err != nil {
		return "", errors.Annotate(err, "cros image name from usb drive").Err()
	}
	var isTestImage bool
	var imageName string
	for _, l := range strings.Split(out, "\n") {
		if imageName != "" && isTestImage {
			break
		}
		if !isTestImage && crosTestImageTrack.MatchString(l) {
			isTestImage = true
			log.Info("ChromeOS image name from USB drive: that is test image: %s", l)
			continue
		}
		if re := crosTestImageName.FindStringSubmatch(l); len(re) > 1 {
			imageName = re[1]
			log.Info("ChromeOS image name from USB drive: image name: %q", imageName)
			continue
		}
	}
	if !isTestImage {
		return "", errors.Reason("cros image name from usb drive: is not test image").Err()
	}
	if imageName == "" {
		return "", errors.Reason("cros image name from usb drive: image name not found").Err()
	}
	return imageName, nil
}
