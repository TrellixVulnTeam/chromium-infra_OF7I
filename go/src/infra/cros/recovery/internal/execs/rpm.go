// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/tlw"
)

// RPMAction perfrom RPM action on RPM outlet.
// If the outlet missing the state will set to missing config state.
func (a *RunArgs) RPMAction(ctx context.Context, hostname string, o *tlw.RPMOutlet, action tlw.RunRPMActionRequest_Action) error {
	if o.GetHostname() == "" || o.GetOutlet() == "" {
		o.State = tlw.RPMOutlet_MISSING_CONFIG
		return errors.Reason("rpm action: missing outlet config").Err()
	}
	req := &tlw.RunRPMActionRequest{
		Hostname:    hostname,
		RpmHostname: o.GetHostname(),
		RpmOutlet:   o.GetOutlet(),
		Action:      action,
	}
	err := a.Access.RunRPMAction(ctx, req)
	return errors.Annotate(err, "rpm action").Err()
}
