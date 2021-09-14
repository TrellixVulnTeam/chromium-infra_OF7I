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
)

// provisionExec performs provisioning of the device.
//
// To prevent reboot of device please provide action exec argument 'no_reboot'.
func provisionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	preventReboot := false
	for _, arg := range actionArgs {
		if arg == "no_reboot" {
			preventReboot = true
		}
	}
	if preventReboot {
		log.Debug(ctx, "Provision will be perform without reboot.")
	}
	gsPath := fmt.Sprintf("%s/%s", gsCrOSImageBucket, args.DUT.StableVersion.CrosImage)
	log.Debug(ctx, "Provision using image: %s", gsPath)
	return errors.Reason("provision: not implemented").Err()
}

func init() {
	execs.Register("cros_provision", provisionExec)
}
