// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

const (
	// Default reboot command for ChromeOS devices.
	// Each command set sleep 1 second to wait for reaction of the command from left part.
	rebootCommand = "(echo begin 1; sync; echo end 1 \"$?\")& sleep 1;" +
		"(echo begin 2; reboot; echo end 2 \"$?\")& sleep 1;" +
		// Force reboot is not calling shutdown.
		"(echo begin 3; reboot -f; echo end 3 \"$?\")& sleep 1;" +
		// Force reboot without sync.
		"(echo begin 4; reboot -nf; echo end 4 \"$?\")& sleep 1;" +
		// telinit 6 sets run level for process initialized, which is equivalent to reboot.
		"(echo begin 5; telinit 6; echo end 5 \"$?\")"
)

// Reboot executes the reboot command using a command runner for a
// DUT.
//
// This function executes an ellaborate reboot sequence that includes
// executing sync and then attempting forcible reboot etc.
func Reboot(ctx context.Context, run execs.Runner, timeout time.Duration) error {
	log.Debugf(ctx, "Reboot Helper : %s", rebootCommand)
	out, err := run(ctx, timeout, rebootCommand)
	if execs.NoExitStatusErrorInternal.In(err) {
		// Client closed connected as rebooting.
		log.Debugf(ctx, "Client exit as device rebooted: %s", err)
	} else if err != nil {
		return errors.Annotate(err, "reboot helper").Err()
	}
	log.Debugf(ctx, "Stdout: %s", out)
	return nil
}
