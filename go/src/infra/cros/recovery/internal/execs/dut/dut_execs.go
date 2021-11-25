// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

// hasDutNameActionExec verifies that DUT provides name.
func hasDutNameActionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if args.DUT != nil && args.DUT.Name != "" {
		log.Debug(ctx, "DUT name: %q", args.DUT.Name)
		return nil
	}
	return errors.Reason("dut name is empty").Err()
}

// hasDutBoardActionExec verifies that DUT provides board name.
func hasDutBoardActionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if args.DUT != nil && args.DUT.Board != "" {
		log.Debug(ctx, "DUT board name: %q", args.DUT.Board)
		return nil
	}
	return errors.Reason("dut board name is empty").Err()
}

// hasDutModelActionExec verifies that DUT provides model name.
func hasDutModelActionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if args.DUT != nil && args.DUT.Model != "" {
		log.Debug(ctx, "DUT model name: %q", args.DUT.Model)
		return nil
	}
	return errors.Reason("dut model name is empty").Err()
}

// hasDutDeviceSkuActionExec verifies that DUT has the device sku label.
func hasDutDeviceSkuActionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	deviceSkuLabel := args.DUT.DeviceSku
	if deviceSkuLabel == "" {
		return errors.Reason("dut device sku label is empty").Err()
	}
	log.Debug(ctx, "dut device sku label: %s.", deviceSkuLabel)
	return nil
}

func init() {
	execs.Register("has_dut_name", hasDutNameActionExec)
	execs.Register("has_dut_board_name", hasDutBoardActionExec)
	execs.Register("has_dut_model_name", hasDutModelActionExec)
	execs.Register("has_dut_device_sku", hasDutDeviceSkuActionExec)
}
