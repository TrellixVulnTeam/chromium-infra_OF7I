// Copyright 2022 The Chromium OS Authors. All rights reserved.  Use
// of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/internal/log"
)

// TODO: (vkjoshi@) These constants are repeated from the package
// "internal/execs/servo". Over a period of time, all such utilities
// that help exec functions will be migrated within components. Bug
// b/222941834 will track this.
const (
	// servodPdRoleCmd is the servod command for
	servodPdRoleCmd = "servo_pd_role"
	// servodPdRoleValueSnk is the one of the two values for servodPdRoleCmd
	// snk is the state that servo in normal mode and not passes power to DUT.
	servodPdRoleValueSnk = "snk"
	// servodPdRoleValueSrc is the one of the two values for servodPdRoleCmd
	// src is the state that servo in power delivery mode and passes power to the DUT.
	servodPdRoleValueSrc = "src"
)

func ServodPdRoleCmd() string {
	return servodPdRoleCmd
}

func ServodPdRoleValueSnk() string {
	return servodPdRoleValueSnk
}

func ServodPdRoleValueSrc() string {
	return servodPdRoleValueSrc
}

type PDRole struct {
	role string
}

var (
	PD_ON  = PDRole{ServodPdRoleValueSrc()}
	PD_OFF = PDRole{ServodPdRoleValueSnk()}
)

// SetPDRole sets the power-delivery role for servo to the passed
// role-value if the power-delivery control is supported by servod.
func SetPDRole(ctx context.Context, servod components.Servod, role PDRole, pd_required bool) error {
	if err := servod.Has(ctx, ServodPdRoleCmd()); err != nil {
		msg := fmt.Sprintf("control %q is not supported, cannot set target role %q", ServodPdRoleCmd(), role)
		log.Infof(ctx, "Set PD Role: %q", msg)
		if pd_required {
			log.Debugf(ctx, "Set PD Role: PD is not supported by this servo, but is required.")
			return errors.Reason("set PD role: %q", msg).Err()
		}
		return nil
	}
	currentRole, err := GetString(ctx, servod, ServodPdRoleCmd())
	if err != nil {
		log.Debugf(ctx, "Set PD Role: could not determine current PD role")
		errors.Annotate(err, "set PD role").Err()
	}
	currentPDRole := PDRole{currentRole}
	if currentPDRole == role {
		log.Debugf(ctx, "Set PD Role: PD role is already %q", role)
		return nil
	}
	if err := servod.Set(ctx, ServodPdRoleCmd(), role.role); err != nil {
		log.Debugf(ctx, "Set PD Role: %q", err.Error())
		return errors.Annotate(err, "set PD role").Err()
	}
	return nil
}
