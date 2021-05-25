// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package harness

// Option is passed to Open to configure the harness.
// There can be two level of options, one directly applies to
// Info while another one will applies to each DUTHarness.
type Option interface {
	option()
}

type dutHarnessOption interface {
	Option
	configureDutHarness(*DUTHarness)
}

// Assert that updateInventoryOpt matches dutHarnessOption
var _ dutHarnessOption = updateInventoryOpt{}

type updateInventoryOpt struct {
	name string
}

func (updateInventoryOpt) option() {}

func (o updateInventoryOpt) configureDutHarness(dh *DUTHarness) {
	dh.labelUpdater.TaskName = o.name
	dh.labelUpdater.UpdateLabels = true
}

// UpdateInventory returns an updateInventoryOpt that enables
// inventory updates. A task name to be associated with the
// inventory update should be provided.
func UpdateInventory(name string) Option {
	return updateInventoryOpt{name: name}
}
