// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"infra/libs/skylab/inventory"
)

func init() {
	converters = append(converters, callboxConverter)
	reverters = append(reverters, callboxReverter)
}

func callboxConverter(dims Dimensions, ls *inventory.SchedulableLabels) {
	if ls.GetCallbox() {
		dims["label-callbox"] = []string{"True"}
	}
}

func callboxReverter(ls *inventory.SchedulableLabels, d Dimensions) Dimensions {
	d = assignLastBoolValueAndDropKey(d, ls.Callbox, "label-callbox")
	return d
}
