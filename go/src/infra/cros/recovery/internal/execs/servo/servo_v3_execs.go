// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

// servoIsV3Exec checks if the DUT's servo-host is version V3.
func servoIsV3Exec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if !strings.Contains(args.DUT.ServoHost.Name, "-servo") {
		return errors.Reason("servo is not v3").Err()
	}
	log.Info(ctx, `Using servo V3.`)
	return nil
}

func init() {
	execs.Register("servo_is_v3", servoIsV3Exec)
}
