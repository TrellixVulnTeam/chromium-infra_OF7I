// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	"go.chromium.org/luci/common/logging"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/util"
)

// ImportStates imports states of UFS resources.
func ImportStates(ctx context.Context, machines []*crimson.Machine, vms []*crimson.VM, pageSize int) (*datastore.OpResults, error) {
	states := make([]*ufspb.StateRecord, 0)
	logging.Debugf(ctx, "collecting states of machines")
	for _, m := range machines {
		states = append(states, &ufspb.StateRecord{
			ResourceName: util.AddPrefix("machine", m.GetName()),
			State:        util.ToState(m.GetState()),
			User:         util.DefaultImporter,
		})
	}
	logging.Debugf(ctx, "collecting states of vms")
	for _, vm := range vms {
		states = append(states, &ufspb.StateRecord{
			ResourceName: util.AddPrefix("vm", vm.GetName()),
			State:        util.ToState(vm.GetState()),
			User:         util.DefaultImporter,
		})
	}
	return nil, nil
}
