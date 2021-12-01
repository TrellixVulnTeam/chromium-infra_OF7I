// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package labels

import "infra/libs/skylab/inventory"

func init() {
	converters = append(converters, callboxConverter)
	reverters = append(reverters, callboxReverter)
}

func callboxConverter(ls *inventory.SchedulableLabels) []string {
	if ls.GetCallbox() {
		return []string{"callbox"}
	}
	return nil
}

func callboxReverter(ls *inventory.SchedulableLabels, labels []string) []string {
	for i, label := range labels {
		if label != "callbox" {
			continue
		}
		*ls.Callbox = true
		labels = removeLabel(labels, i)
		break
	}
	return labels
}
