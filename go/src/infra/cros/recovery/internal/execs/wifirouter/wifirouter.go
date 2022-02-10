// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package wifirouter

import (
	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/tlw"
)

// activeHost finds active host related to the executed plan.
func activeHost(args *execs.RunArgs) (*tlw.WifiRouterHost, error) {
	for _, router := range args.DUT.WifiRouterHosts {
		if router.GetName() == args.ResourceName {
			return router, nil
		}
	}
	return nil, errors.Reason("router: router not found").Err()
}
