// Copyright 2022 The Chromium OS Authors. All rights reserved.  Use
// of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/internal/log"
)

// SetEcUartCmd will set "ec_uart_cmd" to the specific value based on the passed in parameter.
// Before and after the set of the "ec_uart_cmd", it will toggle the value of "ec_uart_flush".
func SetEcUartCmd(ctx context.Context, servod components.Servod, value string) error {
	const ecUartFlush = "ec_uart_flush"
	log.Info(ctx, `Setting servod command %q to "off" value.`, ecUartFlush)
	if err := servod.Set(ctx, ecUartFlush, "off"); err != nil {
		return errors.Annotate(err, "set ec uart cmd").Err()
	}
	const ecUartCmd = "ec_uart_cmd"
	log.Info(ctx, `Setting servod command %q to %q value.`, ecUartCmd, value)
	if err := servod.Set(ctx, ecUartCmd, value); err != nil {
		return errors.Annotate(err, "set ec uart cmd").Err()
	}
	log.Info(ctx, `Setting servod command %q to "on" value.`, ecUartFlush)
	if err := servod.Set(ctx, ecUartFlush, "on"); err != nil {
		return errors.Annotate(err, "set ec uart cmd").Err()
	}
	return nil
}
