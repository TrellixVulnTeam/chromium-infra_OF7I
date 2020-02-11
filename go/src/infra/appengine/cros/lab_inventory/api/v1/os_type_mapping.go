// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package api

import (
	"infra/libs/skylab/inventory"
)

var boardToOsTypeMapping = map[string]inventory.SchedulableLabels_OSType{
	"fizz-moblab": inventory.SchedulableLabels_OS_TYPE_MOBLAB,
	"gale":        inventory.SchedulableLabels_OS_TYPE_JETSTREAM,
	"mistral":     inventory.SchedulableLabels_OS_TYPE_JETSTREAM,
	"whirlwind":   inventory.SchedulableLabels_OS_TYPE_JETSTREAM,
}
