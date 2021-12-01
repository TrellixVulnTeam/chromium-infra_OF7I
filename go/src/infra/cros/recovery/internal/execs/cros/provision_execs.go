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
	req := &tlw.ProvisionRequest{
		Resource:      args.ResourceName,
		PreventReboot: false,
	}
	argsMap := execs.ParseActionArgs(ctx, actionArgs, execs.DefaultSplitter)
	if _, ok := argsMap["no_reboot"]; ok {
		req.PreventReboot = true
		log.Debug(ctx, "Cros provision will be perform without reboot.")
	}
	osImageName := args.DUT.StableVersion.CrosImage
	if image, ok := argsMap["os_name"]; ok {
		osImageName = image
		log.Debug(ctx, "Cros provision received custom OS image name: %s", osImageName)
	}
	osImageBucket := gsCrOSImageBucket
	if bucket, ok := argsMap["os_bucket"]; ok {
		osImageBucket = bucket
		log.Debug(ctx, "Cros provision received custom OS bucket name: %s", osImageBucket)
	}
	osImagePath := fmt.Sprintf("%s/%s", osImageBucket, osImageName)
	if imagePath, ok := argsMap["os_image_path"]; ok {
		osImagePath = imagePath
		log.Debug(ctx, "Cros provision received custom OS image path: %s", osImagePath)
	}
	req.SystemImagePath = osImagePath
	log.Debug(ctx, "Cros provision OS image path: %s", req.SystemImagePath)
	err := args.Access.Provision(ctx, req)
	return errors.Annotate(err, "cros provision").Err()
}

func init() {
	execs.Register("cros_provision", provisionExec)
}
