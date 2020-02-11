// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package labels

import (
	"infra/libs/skylab/inventory"
)

func init() {
	converters = append(converters, hwidCompConverter)
	reverters = append(reverters, hwidCompReverter)
}

const hwidComponentName = "hwid_component"

func hwidCompConverter(ls *inventory.SchedulableLabels) []string {
	var labels []string
	for _, v := range ls.GetHwidComponent() {
		if v != "" {
			lv := hwidComponentName + ":" + v
			labels = append(labels, lv)
		}
	}
	return labels
}

func hwidCompReverter(ls *inventory.SchedulableLabels, labels []string) []string {
	ls.HwidComponent = nil
	for i := 0; i < len(labels); i++ {
		if k, v := splitLabel(labels[i]); k == hwidComponentName {
			ls.HwidComponent = append(ls.HwidComponent, v)
			labels = removeLabel(labels, i)
			i--
		}
	}
	return labels
}
