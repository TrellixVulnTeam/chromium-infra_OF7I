// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"

	"go.chromium.org/luci/common/errors"
)

// samplePassActionExec provides example to run action which always pass.
func samplePassActionExec(ctx context.Context, args *RunArgs) error {
	return nil
}

// sampleFailActionExec provides example to run action which always fail.
func sampleFailActionExec(ctx context.Context, args *RunArgs) error {
	return errors.Reason("failed").Err()
}

func init() {
	execMap["sample_pass"] = samplePassActionExec
	execMap["sample_fail"] = sampleFailActionExec
}
