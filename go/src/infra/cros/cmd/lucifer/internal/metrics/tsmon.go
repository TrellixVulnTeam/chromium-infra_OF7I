// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package metrics

import "go.chromium.org/luci/common/tsmon/metric"

var (
	// StartCounter is a tsmon counter for lucifer start events.
	StartCounter = metric.NewCounter(
		"chromeos/lucifer/run_job/start",
		"lucifer start events",
		nil)
)
