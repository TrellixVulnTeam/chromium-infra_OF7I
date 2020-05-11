// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"strings"

	fleet "infra/unifiedfleet/api/v1/proto"

	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"
	"go.chromium.org/luci/machine-db/api/crimson/v1"
)

// ToChromeMachines converts crimson machines to UFS format.
func ToChromeMachines(old []*crimson.Machine) []*fleet.Machine {
	newObjects := make([]*fleet.Machine, len(old))
	for i, o := range old {
		newObjects[i] = &fleet.Machine{
			// Temporarily use existing display name as browser machine's name instead of serial number/assettag
			Name:     o.Name,
			Location: toLocation(o.Rack, o.Datacenter),
			Device: &fleet.Machine_ChromeBrowserMachine{
				ChromeBrowserMachine: &fleet.ChromeBrowserMachine{
					DisplayName:    o.Name,
					ChromePlatform: o.Platform,
					// TODO(xixuan): add Nic, KvmInterface, RpmInterface, NetworkDeviceInterface, Drac later
					DeploymentTicket: o.DeploymentTicket,
				},
			},
		}
	}
	return newObjects
}

func toLocation(rack, datacenter string) *fleet.Location {
	l := &fleet.Location{
		Rack: rack,
	}
	switch strings.ToLower(datacenter) {
	case "atl97":
		l.Lab = fleet.Lab_LAB_DATACENTER_ATL97
	case "iad97":
		l.Lab = fleet.Lab_LAB_DATACENTER_IAD97
	case "mtv96":
		l.Lab = fleet.Lab_LAB_DATACENTER_MTV96
	case "mtv97":
		l.Lab = fleet.Lab_LAB_DATACENTER_MTV97
	case "lab01":
		l.Lab = fleet.Lab_LAB_DATACENTER_FUCHSIA
	default:
		l.Lab = fleet.Lab_LAB_UNSPECIFIED
	}
	return l
}

// ToChromePlatforms converts platforms in static file to UFS format.
func ToChromePlatforms(oldP *crimsonconfig.Platforms) []*fleet.ChromePlatform {
	ps := oldP.GetPlatform()
	newP := make([]*fleet.ChromePlatform, len(ps))
	for i, p := range ps {
		newP[i] = &fleet.ChromePlatform{
			Name:         p.GetName(),
			Manufacturer: p.GetManufacturer(),
			Description:  p.GetDescription(),
		}
	}
	return newP
}
