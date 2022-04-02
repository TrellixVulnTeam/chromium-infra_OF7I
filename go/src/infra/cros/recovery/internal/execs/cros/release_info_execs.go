// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components/cros"
	"infra/cros/recovery/internal/execs"
)

// isOSTestChannelExec check if device OS is on testimage-channel.
func isOSTestChannelExec(ctx context.Context, info *execs.ExecInfo) error {
	argsMap := info.GetActionArgs(ctx)
	log := info.NewLogger()
	run := info.DefaultRunner()
	expected := argsMap.AsString(ctx, "channel", "testimage-channel")
	log.Debugf("Channel expected: %s", expected)
	fromDevice, err := cros.ReleaseTrack(ctx, run, log)
	if err != nil {
		return errors.Annotate(err, "is OS on test channel").Err()
	}
	log.Debugf("Channel from device: %s", fromDevice)
	if fromDevice != expected {
		return errors.Reason("is OS on test channel: channels mismatch, expected %q, found %q", expected, fromDevice).Err()
	}
	return nil
}

func init() {
	execs.Register("cros_is_os_test_channel", isOSTestChannelExec)
}
