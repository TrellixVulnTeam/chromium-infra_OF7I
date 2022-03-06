// Copyright 2022 The Chromium OS Authors. All rights reserved.  Use
// of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
)

// ServodGetString retrieves from servod the value of servod command
// passed as an argument, and returns it as a string.
func ServodGetString(ctx context.Context, servod components.Servod, command string) (string, error) {
	// TODO: (vkjoshi@): this function is being moved from the package
	// internal/execs/servo to internal/components/servo. Eventually
	// all the uses old method will be updated to the usage of this
	// new method. But b/222941834 will track this task.
	res, err := servod.Get(ctx, command)
	if err != nil {
		return "", errors.Annotate(err, "servod get").Err()
	}
	return res.GetString_(), nil
}
