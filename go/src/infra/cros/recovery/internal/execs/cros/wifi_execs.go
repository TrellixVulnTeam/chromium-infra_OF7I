// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

const (
	// command to check whether the wifi device has been recogonized
	// and its device driver been loaded by the kernel.
	wifiDetectCmd = `lspci -vvn | grep iwlwifi`
)

// auditWiFiExec will validate wifi chip and update state.
//
// Detect if the DUT has wifi device listed in the output of 'lspci' command.
func auditWiFiExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.NewRunner(args.ResourceName)
	_, err := r(ctx, time.Minute, wifiDetectCmd)
	if err == nil {
		// successfully detected
		args.DUT.Wifi.State = tlw.HardwareStateNormal
		log.Info(ctx, "set wifi state to be: %s", tlw.HardwareStateNormal)
		return nil
	}
	if execs.SSHErrorInternal.In(err) || execs.SSHErrorCLINotFound.In(err) {
		args.DUT.Wifi.State = tlw.HardwareStateUnspecified
		return errors.Annotate(err, "audit wifi").Err()
	}
	if args.DUT.Wifi.ChipName != "" {
		// If wifi chip is not detected, but was expected by setup info then we
		// set needs_replacement as it is probably a hardware issue.
		args.DUT.Wifi.State = tlw.HardwareStateNeedReplacement
		log.Info(ctx, "set wifi state to be: %s", tlw.HardwareStateNeedReplacement)
		return errors.Annotate(err, "audit wifi").Err()
	}
	// the wifi state cannot be determined due to cmd failed
	// therefore, set it to HardwareStateNotDetected
	args.DUT.Wifi.State = tlw.HardwareStateNotDetected
	log.Info(ctx, "set wifi state to be: %s", tlw.HardwareStateNotDetected)
	return errors.Annotate(err, "audit wifi").Err()
}

func init() {
	execs.Register("cros_audit_wifi", auditWiFiExec)
}
