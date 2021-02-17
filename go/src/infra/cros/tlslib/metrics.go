// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tlslib

import (
	"go.chromium.org/luci/common/tsmon"
	"go.chromium.org/luci/common/tsmon/metric"
	"go.chromium.org/luci/common/tsmon/target"
)

const taskName = "cros-tls"

var (
	provisionDutCounter = metric.NewCounter(
		"chromeos/tls/provisiondut/count",
		"TLS ProvisionDut counter",
		nil)
)

func newTsmonFlags() *tsmon.Flags {
	fl := tsmon.NewFlags()
	fl.Flush = tsmon.FlushAuto
	fl.Target.SetDefaultsFromHostname()
	fl.Target.TargetType = target.TaskType
	fl.Target.TaskServiceName = taskName
	fl.Target.TaskJobName = taskName
	return &fl
}
