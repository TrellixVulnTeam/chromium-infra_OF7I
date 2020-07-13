// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dumper

import (
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
)

var (
	dumpToBQTick = metric.NewCounter(
		"chromeos/ufs/dumper/dump_to_bq",
		"dumpToBQ attempt",
		nil,
		field.Bool("success"), // If the attempt succeed
	)
	dumpCrosInvcentoryTick = metric.NewCounter(
		"chromeos/ufs/dumper/import_cros_inventory",
		"import cros inventory attempt",
		nil,
		field.Bool("success"),
	)
	dumpCrosNetworkTick = metric.NewCounter(
		"chromeos/ufs/dumper/import_cros_network",
		"import cros network attempt",
		nil,
		field.Bool("success"),
	)
)
