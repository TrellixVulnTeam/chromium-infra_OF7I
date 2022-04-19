// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package linux

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
)

const (
	// Minimum time  to execute command on the hosts as 30 seconds.
	minRunTimeout = 30 * time.Second
)

// MountDrive mounts an external drive on host.
func MountDrive(ctx context.Context, run components.Runner, mountPath, srcPath string) error {
	if _, err := run(ctx, minRunTimeout, "mkdir", "-p", mountPath); err != nil {
		return errors.Annotate(err, "mount drive").Err()
	}
	if _, err := run(ctx, minRunTimeout, "mount", "-o", "ro", srcPath, mountPath); err != nil {
		return errors.Annotate(err, "mount drive %q", srcPath).Err()
	}
	return nil
}

// UnmountDrive unmounts a drive from host.
func UnmountDrive(ctx context.Context, run components.Runner, mountPath string) error {
	_, err := run(ctx, minRunTimeout, "umount", mountPath)
	return errors.Annotate(err, "unmount drive %q", mountPath).Err()
}
