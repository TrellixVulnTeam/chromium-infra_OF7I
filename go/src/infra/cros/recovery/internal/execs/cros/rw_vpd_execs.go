// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

// RwVPDMap are the default requared values for RW_VPD
var RwVPDMap = map[string]string{
	"should_send_rlz_ping": "0",
	"gbind_attribute":      "=CikKIKxeOtv7AqiCHCDBHkyLN-HF0S7JRcZgsoIRvkPlfMqaEAAaA2V2ZRCX0O3NBg==",
	"ubind_attribute":      "=CikKILiLqJanLAzsXFuVPmfc_aOZxnyNirT9iesdM6kt59x6EAEaA2V2ZRCIkb3GCQ==",
}

const (
	// readRwVPDValuesCmdGlob reads the value of VPD key from RW_VPD partition by name.
	readRwVPDValuesCmdGlob = "vpd -i RW_VPD -g %s"
)

// areRequiredRWVPDKeysPresentExec confirms that there is no required RW_VPD keys missing on the device.
func areRequiredRWVPDKeysPresentExec(ctx context.Context, info *execs.ExecInfo) error {
	r := info.DefaultRunner()
	for k := range RwVPDMap {
		cmd := fmt.Sprintf(readRwVPDValuesCmdGlob, k)
		if _, err := r(ctx, time.Minute, cmd); err != nil {
			return errors.Annotate(err, "any rw vpd keys missing").Err()
		}
	}
	log.Infof(ctx, "no rw_vpd values missing")
	return nil
}

func init() {
	execs.Register("cros_are_required_rw_vpd_keys_present", areRequiredRWVPDKeysPresentExec)
}
