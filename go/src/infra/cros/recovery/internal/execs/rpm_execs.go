// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/tlw"
)

// hasRpmInfoActionExec verifies if rpm info is present for DUT.
func hasRpmInfoActionExec(ctx context.Context, args *RunArgs) error {
	if args.DUT.RPMOutlet != nil {
		name := args.DUT.RPMOutlet.Name
		// TODO(otabek@): set fixed number to check and add accept argument value.
		if name != "" && strings.Contains(name, "|") {
			return nil
		}
	}
	return errors.Reason("has rpm info: not present or incorrect").Err()
}

// rpmPowerCycleActionExec performs power cycle the device by RPM.
func rpmPowerCycleActionExec(ctx context.Context, args *RunArgs) error {
	req := &tlw.SetPowerSupplyRequest{
		Resource: args.DUT.Name,
		State:    tlw.PowerSupplyActionCycle,
	}
	res := args.Access.SetPowerSupply(ctx, req)
	switch res.Status {
	case tlw.PowerSupplyResponseStatusOK:
		return nil
	case tlw.PowerSupplyResponseStatusNoConfig:
		if args.DUT.RPMOutlet != nil {
			args.DUT.RPMOutlet.State = tlw.RPMStateMissingConfig
		}
		return errors.Reason("rpm power cycle: no config: %s", res.Reason).Err()
	case tlw.PowerSupplyResponseStatusBadRequest:
		return errors.Reason("rpm power cycle: bad request: %s", res.Reason).Err()
	case tlw.PowerSupplyResponseStatusError:
		return errors.Reason("rpm power cycle: %s", res.Reason).Err()
	default:
		return errors.Reason("rpm power cycle: unexpected status").Err()
	}
}

func init() {
	execMap["has_rpm_info"] = hasRpmInfoActionExec
	execMap["rpm_power_cycle"] = rpmPowerCycleActionExec
}
