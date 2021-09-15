// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpm

import (
	"context"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

// hasRpmInfoExec verifies if rpm info is present for DUT.
func hasRpmInfoExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if args.DUT.RPMOutlet != nil {
		name := args.DUT.RPMOutlet.Name
		// TODO(otabek@): set fixed number to check and add accept argument value.
		if name != "" && strings.Contains(name, "|") {
			return nil
		}
		args.DUT.RPMOutlet.State = tlw.RPMStateMissingConfig
	}
	return errors.Reason("has rpm info: not present or incorrect").Err()
}

// rpmPowerCycleExec performs power cycle the device by RPM.
func rpmPowerCycleExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	req := &tlw.SetPowerSupplyRequest{
		Resource: args.DUT.Name,
		State:    tlw.PowerSupplyActionCycle,
	}
	res := args.Access.SetPowerSupply(ctx, req)
	switch res.Status {
	case tlw.PowerSupplyResponseStatusOK:
		log.Debug(ctx, "RPM power cycle finished with success.")
		return nil
	case tlw.PowerSupplyResponseStatusNoConfig:
		args.DUT.RPMOutlet.State = tlw.RPMStateMissingConfig
		return errors.Reason("rpm power cycle: no config: %s", res.Reason).Err()
	case tlw.PowerSupplyResponseStatusBadRequest:
		return errors.Reason("rpm power cycle: bad request: %s", res.Reason).Err()
	case tlw.PowerSupplyResponseStatusError:
		return errors.Reason("rpm power cycle: %s", res.Reason).Err()
	default:
		return errors.Reason("rpm power cycle: unexpected status").Err()
	}
}

// rpmPowerOffExec performs power off the device by RPM.
func rpmPowerOffExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	req := &tlw.SetPowerSupplyRequest{
		Resource: args.DUT.Name,
		State:    tlw.PowerSupplyActionOff,
	}
	res := args.Access.SetPowerSupply(ctx, req)
	switch res.Status {
	case tlw.PowerSupplyResponseStatusOK:
		log.Debug(ctx, "RPM power OFF finished with success.")
		return nil
	case tlw.PowerSupplyResponseStatusNoConfig:
		args.DUT.RPMOutlet.State = tlw.RPMStateMissingConfig
		return errors.Reason("rpm power off: no config: %s", res.Reason).Err()
	case tlw.PowerSupplyResponseStatusBadRequest:
		return errors.Reason("rpm power off: bad request: %s", res.Reason).Err()
	case tlw.PowerSupplyResponseStatusError:
		return errors.Reason("rpm power off: %s", res.Reason).Err()
	default:
		return errors.Reason("rpm power off: unexpected status").Err()
	}
}

// rpmPowerOffExec performs power on the device by RPM.
func rpmPowerOnExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	req := &tlw.SetPowerSupplyRequest{
		Resource: args.DUT.Name,
		State:    tlw.PowerSupplyActionOn,
	}
	res := args.Access.SetPowerSupply(ctx, req)
	switch res.Status {
	case tlw.PowerSupplyResponseStatusOK:
		log.Debug(ctx, "RPM power ON finished with success.")
		return nil
	case tlw.PowerSupplyResponseStatusNoConfig:
		args.DUT.RPMOutlet.State = tlw.RPMStateMissingConfig
		return errors.Reason("rpm power on: no config: %s", res.Reason).Err()
	case tlw.PowerSupplyResponseStatusBadRequest:
		return errors.Reason("rpm power on: bad request: %s", res.Reason).Err()
	case tlw.PowerSupplyResponseStatusError:
		return errors.Reason("rpm power on: %s", res.Reason).Err()
	default:
		return errors.Reason("rpm power on: unexpected status").Err()
	}
}

func init() {
	execs.Register("has_rpm_info", hasRpmInfoExec)
	execs.Register("rpm_power_cycle", rpmPowerCycleExec)
	execs.Register("rpm_power_off", rpmPowerOffExec)
	execs.Register("rpm_power_on", rpmPowerOnExec)
}
