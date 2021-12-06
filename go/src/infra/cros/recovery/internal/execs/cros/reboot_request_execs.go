// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

const (
	// Filter used to find reboot request flags files.
	// Examples: '/var/lib/servod/somebody_reboot'
	rebootRequestFindCmd          = "find /var/lib/servod/*_reboot"
	rebootRequestRemoveAllCmd     = "rm /var/lib/servod/*_reboot"
	rebootRequestCreateSingleGlob = "touch /var/lib/servod/%d_reboot"
	rebootRequestRemoveSingleGlob = "rm /var/lib/servod/%d_reboot"
)

// createRebootRequestExec creates reboot flag file request.
func createRebootRequestExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.Access.Run(ctx, args.ResourceName, fmt.Sprintf(rebootRequestCreateSingleGlob, args.DUT.ServoHost.ServodPort))
	// Print finish result as we ignore any errors.
	log.Debug(ctx, "Create the reboot request: finished with code: %d, error: %s", r.ExitCode, r.Stderr)
	return nil
}

// hasRebootRequestExec checks presence of reboot request flag on the host.
func hasRebootRequestExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.Access.Run(ctx, args.ResourceName, rebootRequestFindCmd)
	// Print finish result as we treat failure as no results.
	log.Debug(ctx, "Has reboot requests: finished with code: %d, error: %s", r.ExitCode, r.Stderr)
	rr := strings.TrimSpace(r.Stdout)
	if rr == "" {
		return errors.Reason("has reboot request: not request found").Err()
	}
	log.Info(ctx, "Found reboot requests:\n%s", rr)
	return nil
}

// removeAllRebootRequestsExec removes all reboot flag file requests.
func removeAllRebootRequestsExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.Access.Run(ctx, args.ResourceName, rebootRequestRemoveAllCmd)
	// Print finish result as we ignore any errors.
	log.Debug(ctx, "Remove all reboot requests: finished with code: %d, error: %s", r.ExitCode, r.Stderr)
	return nil
}

// removeRebootRequestExec removes reboot flag file request.
func removeRebootRequestExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.Access.Run(ctx, args.DUT.ServoHost.Name, fmt.Sprintf(rebootRequestRemoveSingleGlob, args.DUT.ServoHost.ServodPort))
	// Print finish result as we ignore any errors.
	log.Debug(ctx, "Remove the reboot request: finished with code: %d, error: %s", r.ExitCode, r.Stderr)
	return nil
}

func init() {
	execs.Register("cros_create_reboot_request", createRebootRequestExec)
	execs.Register("cros_has_reboot_request", hasRebootRequestExec)
	execs.Register("cros_remove_all_reboot_request", removeAllRebootRequestsExec)
	execs.Register("cros_remove_reboot_request", removeRebootRequestExec)
}
