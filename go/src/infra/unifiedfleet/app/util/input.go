// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"fmt"
	"strings"

	ufspb "infra/unifiedfleet/api/v1/proto"
)

const (
	// MachineCollection refers to the prefix of the corresponding resource.
	MachineCollection string = "machines"
	// RackCollection refers to the prefix of the corresponding resource.
	RackCollection string = "racks"
	// VMCollection refers to the prefix of the corresponding resource.
	VMCollection string = "vms"
	// ChromePlatformCollection refers to the prefix of the corresponding resource.
	ChromePlatformCollection string = "chromeplatforms"
	// MachineLSECollection refers to the prefix of the corresponding resource.
	MachineLSECollection string = "machineLSEs"
	// HostCollection refers to the prefix of the corresponding resource.
	HostCollection string = "hosts"
	// RackLSECollection refers to the prefix of the corresponding resource.
	RackLSECollection string = "rackLSEs"
	// NicCollection refers to the prefix of the corresponding resource.
	NicCollection string = "nics"
	// KVMCollection refers to the prefix of the corresponding resource.
	KVMCollection string = "kvms"
	// RPMCollection refers to the prefix of the corresponding resource.
	RPMCollection string = "rpms"
	// DracCollection refers to the prefix of the corresponding resource.
	DracCollection string = "dracs"
	// SwitchCollection refers to the prefix of the corresponding resource.
	SwitchCollection string = "switches"
	// VlanCollection refers to the prefix of the corresponding resource.
	VlanCollection string = "vlans"
	// MachineLSEPrototypeCollection refers to the prefix of the corresponding resource.
	MachineLSEPrototypeCollection string = "machineLSEPrototypes"
	// RackLSEPrototypeCollection refers to the prefix of the corresponding resource.
	RackLSEPrototypeCollection string = "rackLSEPrototypes"
	// DHCPCollection refers to the prefix of the dhcp config id in change history
	DHCPCollection string = "dhcps"
	// IPCollection refers to the prefix of the ip id in change history
	IPCollection string = "ips"
	// StateCollection refers to the prefix of the states id in change history
	StateCollection string = "states"

	// DefaultImporter refers to the user of the cron job importer
	DefaultImporter string = "crimson-importer"

	defaultPageSize int32 = 100
	// MaxPageSize maximum page size for list operations
	MaxPageSize int32 = 1000
)

// Filter names for indexed properties in datastore for different entities
var (
	ZoneFilterName              string = "zone"
	RackFilterName              string = "rack"
	MachineFilterName           string = "machine"
	HostFilterName              string = "host"
	NicFilterName               string = "nic"
	DracFilterName              string = "drac"
	KVMFilterName               string = "kvm"
	MacAddressFilterName        string = "mac"
	RPMFilterName               string = "rpm"
	SwitchFilterName            string = "switch"
	SwitchPortFilterName        string = "switchport"
	ServoFilterName             string = "servo"
	TagFilterName               string = "tag"
	ChromePlatformFilterName    string = "platform"
	MachinePrototypeFilterName  string = "machineprototype"
	RackPrototypeFilterName     string = "rackprototype"
	VlanFilterName              string = "vlan"
	StateFilterName             string = "state"
	IPV4FilterName              string = "ipv4"
	IPV4StringFilterName        string = "ipv4str"
	OccupiedFilterName          string = "occupied"
	ManufacturerFilterName      string = "man"
	FreeVMFilterName            string = "free"
	ResourceTypeFilterName      string = "resourcetype"
	OSVersionFilterName         string = "osversion"
	OSFilterName                string = "os"
	VirtualDatacenterFilterName string = "vdc"
)

const separator string = "/"

// GetPageSize gets the correct page size for List pagination
func GetPageSize(pageSize int32) int32 {
	switch {
	case pageSize == 0:
		return defaultPageSize
	case pageSize > MaxPageSize:
		return MaxPageSize
	default:
		return pageSize
	}
}

// RemovePrefix extracts string appearing after a "/"
func RemovePrefix(name string) string {
	// Get substring after a string.
	name = strings.TrimSpace(name)
	pos := strings.Index(name, separator)
	if pos == -1 {
		return name
	}
	adjustedPos := pos + len(separator)
	if adjustedPos >= len(name) {
		return name
	}
	return name[adjustedPos:]
}

// AddPrefix adds the prefix for a given resource name
func AddPrefix(collection, entity string) string {
	return fmt.Sprintf("%s%s%s", collection, separator, entity)
}

// GetPrefix returns the prefix for a resource name
func GetPrefix(resourceName string) string {
	s := strings.Split(strings.TrimSpace(resourceName), separator)
	if len(s) < 1 {
		return ""
	}
	return s[0]
}

// GetRackHostname returns a rack host name.
func GetRackHostname(rackName string) string {
	return fmt.Sprintf("%s-host", rackName)
}

// FormatResourceName formats the resource name
func FormatResourceName(old string) string {
	str := strings.Replace(old, " ", "_", -1)
	return strings.Replace(str, ",", "_", -1)
}

// StrToUFSState refers a map between a string to a UFS defined state map.
var StrToUFSState = map[string]string{
	"registered":           "STATE_REGISTERED",
	"deployed_pre_serving": "STATE_DEPLOYED_PRE_SERVING",
	"deployed_testing":     "STATE_DEPLOYED_TESTING",
	"serving":              "STATE_SERVING",
	"needs_reset":          "STATE_NEEDS_RESET",
	"needs_repair":         "STATE_NEEDS_REPAIR",
	"repair_failed":        "STATE_REPAIR_FAILED",
	"disabled":             "STATE_DISABLED",
	"reserved":             "STATE_RESERVED",
	"decommissioned":       "STATE_DECOMMISSIONED",
}

// IsUFSState checks if a string refers to a valid UFS state.
func IsUFSState(state string) bool {
	_, ok := StrToUFSState[state]
	return ok
}

// ValidStateStr returns a valid str list for state strings.
func ValidStateStr() []string {
	ks := make([]string, 0, len(StrToUFSState))
	for k := range StrToUFSState {
		ks = append(ks, k)
	}
	return ks
}

// RemoveStatePrefix removes the "state_" prefix from the string
func RemoveStatePrefix(state string) string {
	state = strings.ToLower(state)
	if idx := strings.Index(state, "state_"); idx != -1 {
		state = state[idx+len("state_"):]
	}
	return state
}

// ToUFSState converts state string to a UFS state enum.
func ToUFSState(state string) ufspb.State {
	state = RemoveStatePrefix(state)
	v, ok := StrToUFSState[state]
	if !ok {
		return ufspb.State_STATE_UNSPECIFIED
	}
	return ufspb.State(ufspb.State_value[v])
}

// StrToUFSZone refers a map between a string to a UFS defined map.
var StrToUFSZone = map[string]string{
	"atlanta":     "ZONE_ATLANTA",
	"chromeos1":   "ZONE_CHROMEOS1",
	"chromeos4":   "ZONE_CHROMEOS4",
	"chromeos6":   "ZONE_CHROMEOS6",
	"chromeos2":   "ZONE_CHROMEOS2",
	"chromeos3":   "ZONE_CHROMEOS3",
	"chromeos5":   "ZONE_CHROMEOS5",
	"chromeos7":   "ZONE_CHROMEOS7",
	"chromeos15":  "ZONE_CHROMEOS15",
	"atl97":       "ZONE_ATL97",
	"iad97":       "ZONE_IAD97",
	"mtv96":       "ZONE_MTV96",
	"mtv97":       "ZONE_MTV97",
	"fuchsia":     "ZONE_FUCHSIA",
	"unspecified": "ZONE_UNSPECIFIED",
}

// IsUFSZone checks if a string refers to a valid UFS zone.
func IsUFSZone(zone string) bool {
	_, ok := StrToUFSZone[zone]
	return ok
}

// ValidZoneStr returns a valid str list for zone strings.
func ValidZoneStr() []string {
	ks := make([]string, 0, len(StrToUFSZone))
	for k := range StrToUFSZone {
		ks = append(ks, k)
	}
	return ks
}

// RemoveZonePrefix removes the "zone_" prefix from the string
func RemoveZonePrefix(zone string) string {
	zone = strings.ToLower(zone)
	if idx := strings.Index(zone, "zone_"); idx != -1 {
		zone = zone[idx+len("zone_"):]
	}
	return zone
}

// ToUFSZone converts zone string to a UFS zone enum.
func ToUFSZone(zone string) ufspb.Zone {
	zone = RemoveZonePrefix(zone)
	v, ok := StrToUFSZone[zone]
	if !ok {
		return ufspb.Zone_ZONE_UNSPECIFIED
	}
	return ufspb.Zone(ufspb.Zone_value[v])
}

// ToUFSRealm returns the realm name based on zone string.
func ToUFSRealm(zone string) string {
	ufsZone := ToUFSZone(zone)
	if IsInBrowserZone(ufsZone.String()) {
		return BrowserLabAdminRealm
	}
	if ufsZone == ufspb.Zone_ZONE_CHROMEOS3 || ufsZone == ufspb.Zone_ZONE_CHROMEOS5 ||
		ufsZone == ufspb.Zone_ZONE_CHROMEOS7 || ufsZone == ufspb.Zone_ZONE_CHROMEOS15 {
		return AcsLabAdminRealm
	}
	return AtlLabAdminRealm
}
