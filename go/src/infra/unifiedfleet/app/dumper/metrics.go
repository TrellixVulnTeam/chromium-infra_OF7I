// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dumper

import (
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
)

var (
	dumpChromePlatformTick = metric.NewCounter(
		"chromeos/ufs/dumper/dump_chrome_platform",
		"dumpChromePlatform attempt",
		nil,
		field.Bool("success"), // If the attempt succeed
	)
)
