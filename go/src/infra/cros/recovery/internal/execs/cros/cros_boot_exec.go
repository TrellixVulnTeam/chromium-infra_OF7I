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

// isBootedFromExternalStorageExec verify that device has been booted from external storage.
func isBootedFromExternalStorageExec(ctx context.Context, info *execs.ExecInfo) error {
	err := cros.IsBootedFromExternalStorage(ctx, info.NewRunner(info.RunArgs.DUT.Name), info.NewLogger())
	return errors.Annotate(err, "is booted from external storage").Err()
}

func init() {
	execs.Register("cros_booted_from_external_storage", isBootedFromExternalStorageExec)
}
