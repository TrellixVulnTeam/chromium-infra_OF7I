// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package api

import (
	"infra/libs/skylab/inventory"
)

// AdaptToV1DutSpec adapts ExtendedDeviceData to inventory.DeviceUnderTest of
// inventory v1.
func AdaptToV1DutSpec(data *ExtendedDeviceData) (*inventory.DeviceUnderTest, error) {
	// TODO (guocb) Implement the adapter.
	return nil, nil
}
