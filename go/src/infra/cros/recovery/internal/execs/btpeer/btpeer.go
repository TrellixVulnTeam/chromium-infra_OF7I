// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package btpeer

import (
	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/tlw"
)

// activeHost finds active host related to the executed plan.
func activeHost(args *execs.RunArgs) (*tlw.BluetoothPeerHost, error) {
	for _, btp := range args.DUT.BluetoothPeerHosts {
		if btp.Name == args.ResourceName {
			return btp, nil
		}
	}
	return nil, errors.Reason("active host: host not found").Err()
}
