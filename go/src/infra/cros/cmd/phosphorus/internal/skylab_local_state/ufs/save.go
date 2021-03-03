// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufs

import (
	"context"

	"infra/cros/dutstate"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// Allowlist of DUT states that are safe to overwrite.
var dutStatesSafeForOverwrite = map[dutstate.State]bool{
	dutstate.NeedsRepair: true,
	dutstate.Ready:       true,
}

// SafeUpdateUFSDUTState attempts to safely update the DUT state to the
// given value in UFS. States other than Ready and NeedsRepair are
// ignored.
func SafeUpdateUFSDUTState(ctx context.Context, authFlags *authcli.Flags, dutName, dutState, ufsService string) error {
	currentDUTState, err := dutStateFromUFS(ctx, authFlags, ufsService, dutName)
	if err != nil {
		return err
	}
	if dutStatesSafeForOverwrite[currentDUTState] {
		return updateDUTStateToUFS(ctx, authFlags, ufsService, dutName, dutState)
	}
	logging.Warningf(ctx, "Not saving requested DUT state %s, since current DUT state is %s, which should never be overwritten", dutState, currentDUTState)
	return nil
}

// updateDUTStateToUFS send DUT state to the UFS service.
func updateDUTStateToUFS(ctx context.Context, authFlags *authcli.Flags, crosUfsService, dutName, dutState string) error {
	ufsClient, err := NewClient(ctx, crosUfsService, authFlags)
	if err != nil {
		return errors.Annotate(err, "save local state").Err()
	}
	err = dutstate.Update(ctx, ufsClient, dutName, dutstate.State(dutState))
	if err != nil {
		return errors.Annotate(err, "save local state").Err()
	}
	return nil
}

// dutStateFromUFS read DUT state from the UFS service.
func dutStateFromUFS(ctx context.Context, authFlags *authcli.Flags, crosUfsService, dutName string) (dutstate.State, error) {
	ufsClient, err := NewClient(ctx, crosUfsService, authFlags)
	if err != nil {
		return "", errors.Annotate(err, "read local state").Err()
	}
	info := dutstate.Read(ctx, ufsClient, dutName)
	logging.Infof(ctx, "Receive DUT state from UFS: %s", info.State)
	return info.State, nil
}
