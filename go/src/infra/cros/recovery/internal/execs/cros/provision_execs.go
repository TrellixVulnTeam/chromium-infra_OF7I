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
func provisionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	req := &tlw.ProvisionRequest{
		Resource:      args.ResourceName,
		PreventReboot: false,
	}
	for _, arg := range actionArgs {
		if arg == "no_reboot" {
			req.PreventReboot = true
			log.Debug(ctx, "Cros provision will be perform without reboot.")
		}
	}
	req.SystemImagePath = fmt.Sprintf("%s/%s", gsCrOSImageBucket, args.DUT.StableVersion.CrosImage)
	log.Debug(ctx, "System image path: %s", req.SystemImagePath)
	err := args.Access.Provision(ctx, req)
	return errors.Annotate(err, "cros provision").Err()
}

func init() {
	execs.Register("cros_provision", provisionExec)
}
