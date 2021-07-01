// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"
	"log"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/retry"
)

// TODO(otabek@): Extract all commands to constants.
// NOTE: That is just fake execs for local testing during developing phase. The correct/final execs will be introduced later.

func servodEchoActionExec(ctx context.Context, args *RunArgs) error {
	res, err := ServodCallGet(ctx, args, "serialname")
	if err != nil {
		return errors.Annotate(err, "servod echo exec").Err()
	} else if res.Value.GetString_() == "" {
		return errors.Reason("servod echo exec: received empty result").Err()
	}
	return nil
}

func servodLidopenActionExec(ctx context.Context, args *RunArgs) error {
	res, err := ServodCallGet(ctx, args, "lid_open")
	if err != nil {
		return errors.Annotate(err, "servod lid_open").Err()
	} else if res.Value.GetString_() == "not_applicable" {
		log.Printf("Device does not support this action. Skipping...")
	} else if res.Value.GetString_() != "yes" {
		return errors.Reason("servod lid_open: expected to received 'yes'").Err()
	}
	return nil
}

func servodLidopenRecoveryActionExec(ctx context.Context, args *RunArgs) error {
	res, err := ServodCallGet(ctx, args, "lid_open")
	if err != nil {
		return errors.Annotate(err, "servod lid_open recovery").Err()
	} else if res.Value.GetString_() == "yes" {
		log.Printf("Servod lid_open recovery: received expected value, skip the recovery execution.")
		return nil
	}
	// Fix is to first try to set `no` then 'yes'. Then verify.
	// TODO(otabek@): Remove when add right execs.
	res, err = ServodCallSet(ctx, args, "lid_open", "no")
	if err != nil {
		return errors.Annotate(err, "servod lid_open recovery").Err()
	}
	res, err = ServodCallSet(ctx, args, "lid_open", "yes")
	if err != nil {
		return errors.Annotate(err, "servod lid_open recovery").Err()
	}
	// Wait 5 seconds to apply effect.
	log.Printf("Servod lid_open recovery: waiting 5 seconds to apply after set lid_open to `yes`.")
	time.Sleep(5 * time.Second)
	res, err = ServodCallGet(ctx, args, "lid_open")
	if err != nil {
		return errors.Annotate(err, "servod lid_open").Err()
	} else if res.Value.GetString_() == "yes" {
		log.Printf("Servod lid_open recovery: fixed")
		return nil
	}
	return errors.Reason("servod lid_open recovery: not able to get expected value 'yes' after toggling lid_open attempt").Err()
}

const (
	// Time to allow for boot from power off. Among other things, this must account for the 30 second dev-mode
	// screen delay, time to start the network on the DUT, and the ssh timeout of 120 seconds.
	dutBootTimeout = 150 * time.Second
	// Time to allow for boot from a USB device, including the 30 second dev-mode delay and time to start the network.
	usbkeyBootTimeout = 300 * time.Second
)

func servodDUTBootRecoveryModeActionExec(ctx context.Context, args *RunArgs) error {
	if _, err := ServodCallSet(ctx, args, "power_state", "rec"); err != nil {
		return errors.Annotate(err, "servod boot in recovery-mode").Err()
	}
	return retry.WithTimeout(ctx, 10*time.Second, usbkeyBootTimeout, func() error {
		if r := args.Access.Run(ctx, args.DUT.Name, "true"); r.ExitCode != 0 {
			return errors.Reason("servod boot in recovery-mode: check ssh access, code: %d, %s", r.ExitCode, r.Stderr).Err()
		}
		return nil
	}, "servod boot in recovery-mode: check ssh access")
}

func servodDUTColdResetActionExec(ctx context.Context, args *RunArgs) error {
	if _, err := ServodCallSet(ctx, args, "power_state", "reset"); err != nil {
		return errors.Annotate(err, "servod cold_reset dut").Err()
	}
	return retry.WithTimeout(ctx, 5*time.Second, dutBootTimeout, func() error {
		return args.Access.Ping(ctx, args.DUT.Name, 2)
	}, "servod cold_reset dut: check ping access")
}

func init() {
	execMap["servod_echo"] = servodEchoActionExec
	execMap["servod_lidopen"] = servodLidopenActionExec
	execMap["servod_lidopen_recover"] = servodLidopenRecoveryActionExec
	execMap["servod_dut_rec_mode"] = servodDUTBootRecoveryModeActionExec
	execMap["servod_dut_cold_reset"] = servodDUTColdResetActionExec
}
