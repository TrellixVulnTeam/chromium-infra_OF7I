// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

// provisionExec performs provisioning of the device.
//
// To prevent reboot of device please provide action exec argument 'no_reboot'.
// To provide custom image data please use 'os_name', 'os_bucket', 'os_image_path'.
func provisionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	argsMap := execs.ParseActionArgs(ctx, actionArgs, execs.DefaultSplitter)
	osImageName := argsMap.AsString(ctx, "os_name", args.DUT.StableVersion.CrosImage)
	log.Debug(ctx, "Used OS image name: %s", osImageName)
	osImageBucket := argsMap.AsString(ctx, "os_bucket", gsCrOSImageBucket)
	log.Debug(ctx, "Used OS bucket name: %s", osImageBucket)
	osImagePath := argsMap.AsString(ctx, "os_image_path", fmt.Sprintf("%s/%s", osImageBucket, osImageName))
	log.Debug(ctx, "Used OS image path: %s", osImagePath)
	req := &tlw.ProvisionRequest{
		Resource:        args.ResourceName,
		PreventReboot:   false,
		SystemImagePath: osImagePath,
	}
	if _, ok := argsMap["no_reboot"]; ok {
		req.PreventReboot = true
		log.Debug(ctx, "Cros provision will be perform without reboot.")
	}
	log.Debug(ctx, "Cros provision OS image path: %s", req.SystemImagePath)
	err := args.Access.Provision(ctx, req)
	return errors.Annotate(err, "cros provision").Err()
}

// Download image to the USB-drive.
//
// To provide custom image data please use 'os_name', 'os_bucket', 'os_image_path'.
func downloadImageToUSBExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	argsMap := execs.ParseActionArgs(ctx, actionArgs, execs.DefaultSplitter)
	osImageName := argsMap.AsString(ctx, "os_name", args.DUT.StableVersion.CrosImage)
	log.Debug(ctx, "Used OS image name: %s", osImageName)
	osImageBucket := argsMap.AsString(ctx, "os_bucket", gsCrOSImageBucket)
	log.Debug(ctx, "Used OS bucket name: %s", osImageBucket)
	osImagePath := argsMap.AsString(ctx, "os_image_path", fmt.Sprintf("%s/%s", osImageBucket, osImageName))
	log.Debug(ctx, "Used OS image path: %s", osImagePath)
	// Requesting convert GC path to caches service path.
	jobRepoURL, err := args.Access.GetCacheUrl(ctx, args.DUT.Name, osImagePath)
	if err != nil {
		return errors.Annotate(err, "download image to usb-drive").Err()
	}
	// Cache service path is not specify the file.
	image := fmt.Sprintf("%s/chromiumos_test_image.bin", jobRepoURL)
	log.Debug(ctx, "Download image for USB-drive: %s", image)
	err = args.NewServod().Set(ctx, "download_image_to_usb_dev", image)
	return errors.Annotate(err, "download image to usb-drive").Err()
}

func init() {
	execs.Register("cros_provision", provisionExec)
	execs.Register("servo_download_image_to_usb", downloadImageToUSBExec)
}
