// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"fmt"

	"go.chromium.org/chromiumos/infra/proto/go/lab"
)

// GetHostname returns the hostname of input ChromeOSDevice.
func GetHostname(d *lab.ChromeOSDevice) string {
	switch t := d.GetDevice().(type) {
	case *lab.ChromeOSDevice_Dut:
		return d.GetDut().GetHostname()
	case *lab.ChromeOSDevice_Labstation:
		return d.GetLabstation().GetHostname()
	default:
		panic(fmt.Sprintf("Unknown device type: %v", t))
	}
}
