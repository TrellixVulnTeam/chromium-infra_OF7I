// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"

	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/datastore"
)

// ImportVMCapacity imports the vm capacity for chrome platform.
func ImportVMCapacity(ctx context.Context, machines []*crimson.Machine, hosts []*crimson.PhysicalHost, pageSize int) (*datastore.OpResults, error) {
	platformToVMCapacity := make(map[string]int32, 0)
	machineToVMCapacity := make(map[string]int32, 0)
	for _, h := range hosts {
		m := h.GetMachine()
		if machineToVMCapacity[m] < h.GetVmSlots() {
			machineToVMCapacity[m] = h.GetVmSlots()
		}
	}
	for _, m := range machines {
		p := m.GetPlatform()
		if platformToVMCapacity[p] < machineToVMCapacity[m.GetName()] {
			platformToVMCapacity[p] = machineToVMCapacity[m.GetName()]
		}
	}
	allRes := make(datastore.OpResults, 0)
	for startToken := ""; ; {
		cps, nextToken, err := configuration.ListChromePlatforms(ctx, int32(pageSize), startToken)
		if err != nil {
			return &allRes, err
		}
		for _, cp := range cps {
			cp.VmCapacity = platformToVMCapacity[cp.GetName()]
		}
		res, err := configuration.ImportChromePlatforms(ctx, cps)
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if nextToken == "" {
			break
		}
		startToken = nextToken
	}
	return &allRes, nil
}
