// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package btpeer

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/tlw"
)

// setStateBrokenExec sets state as BROKEN.
func setStateBrokenExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if h, err := activeHost(args); err != nil {
		return errors.Annotate(err, "set state broken").Err()
	} else {
		h.State = tlw.BluetoothPeerStateBroken
	}
	return nil
}

// setStateWorkingExec sets state as WORKING.
func setStateWorkingExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if h, err := activeHost(args); err != nil {
		return errors.Annotate(err, "set state working").Err()
	} else {
		h.State = tlw.BluetoothPeerStateWorking
	}
	return nil
}

func init() {
	execs.Register("btpeer_state_broken", setStateBrokenExec)
	execs.Register("btpeer_state_working", setStateWorkingExec)
}
