// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/logger"
)

// IsBootedFromExternalStorage verify that device has been booted from external storage.
func IsBootedFromExternalStorage(ctx context.Context, run components.Runner, log logger.Logger) error {
	bootStorage, err := run(ctx, time.Minute, "rootdev", "-s", "-d")
	if err != nil {
		return errors.Annotate(err, "booted from external storage").Err()
	} else if bootStorage == "" {
		return errors.Reason("booted from external storage: booted storage not detected").Err()
	}
	mainStorageCMD := ". /usr/sbin/write_gpt.sh; . /usr/share/misc/chromeos-common.sh; load_base_vars; get_fixed_dst_drive"
	mainStorage, err := run(ctx, time.Minute, mainStorageCMD)
	if err != nil {
		return errors.Annotate(err, "booted from external storage").Err()
	}
	// If main device is not detected then probably it can be dead or broken
	// but as we gt the boot device then it is external one.
	if mainStorage == "" || bootStorage != mainStorage {
		return nil
	}
	return errors.Reason("booted from external storage: booted from main storage").Err()
}
