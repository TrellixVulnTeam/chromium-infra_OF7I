// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package btpeer

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

// setStateBrokenExec sets state as BROKEN.
func setStateBrokenExec(ctx context.Context, info *execs.ExecInfo) error {
	if h, err := activeHost(info.RunArgs); err != nil {
		return errors.Annotate(err, "set state broken").Err()
	} else {
		h.State = tlw.BluetoothPeerStateBroken
	}
	return nil
}

// setStateWorkingExec sets state as WORKING.
func setStateWorkingExec(ctx context.Context, info *execs.ExecInfo) error {
	if h, err := activeHost(info.RunArgs); err != nil {
		return errors.Annotate(err, "set state working").Err()
	} else {
		h.State = tlw.BluetoothPeerStateWorking
	}
	return nil
}

// getDetectedStatusesExec verifies communication with XMLRPC service running on bluetooth-peer and send one request to verify that service is responsive and initialized.
func getDetectedStatusesExec(ctx context.Context, info *execs.ExecInfo) error {
	h, err := activeHost(info.RunArgs)
	if err != nil {
		return errors.Annotate(err, "get detected statuses").Err()
	}
	res, err := Call(ctx, info.RunArgs.Access, h, "GetDetectedStatus")
	if err != nil {
		return errors.Annotate(err, "get detected statuses").Err()
	}
	count := len(res.GetArray().GetValues())
	if count == 0 {
		return errors.Reason("get detected statuses: list is empty").Err()
	}
	log.Debugf(ctx, "Detected statuses count: %v", count)
	return nil
}

func init() {
	execs.Register("btpeer_state_broken", setStateBrokenExec)
	execs.Register("btpeer_state_working", setStateWorkingExec)
	execs.Register("btpeer_get_detected_statuses", getDetectedStatusesExec)
}
