// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
)

// servodGetString retrieves from servod the value of servod command
// passed as an argument, and returns it as a string.
func servodGetString(ctx context.Context, servod components.Servod, command string) (string, error) {
	res, err := servod.Get(ctx, command)
	if err != nil {
		return "", errors.Annotate(err, "servod get").Err()
	}
	return res.GetString_(), nil
}

// servodGetInt retrieves from servod the value of servod command
// passed as an argument, and returns it as a 32-bit integer.
func servodGetInt(ctx context.Context, servod components.Servod, command string) (int32, error) {
	res, err := servod.Get(ctx, command)
	if err != nil {
		return 0, errors.Annotate(err, "servod get").Err()
	}
	return res.GetInt(), nil
}

// servodGetBool retrieves from servod the value of servod command
// passed as an argument, and returns it as boolean.
func servodGetBool(ctx context.Context, servod components.Servod, command string) (bool, error) {
	res, err := servod.Get(ctx, command)
	if err != nil {
		return false, errors.Annotate(err, "servod get").Err()
	}
	return res.GetBoolean(), nil
}

// servodGetDouble retrieves from servod the value of servod command
// passed as an argument, and returns it as 64-bit floating point
// value.
func servodGetDouble(ctx context.Context, servod components.Servod, command string) (float64, error) {
	res, err := servod.Get(ctx, command)
	if err != nil {
		return 0.0, errors.Annotate(err, "servod get").Err()
	}
	return res.GetDouble(), nil
}
