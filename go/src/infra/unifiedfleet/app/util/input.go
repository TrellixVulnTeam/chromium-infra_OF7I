// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
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

	// DefaultImporter refers to the user of the cron job importer
	DefaultImporter string = "crimson-importer"

	defaultPageSize int32 = 100
	maxPageSize     int32 = 1000
)

const separator string = "/"

// GetPageSize gets the correct page size for List pagination
func GetPageSize(pageSize int32) int32 {
	switch {
	case pageSize == 0:
		return defaultPageSize
	case pageSize > maxPageSize:
		return maxPageSize
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

// GetUUIDName returns a new UUID name.
func GetUUIDName() string {
	return uuid.New().String()
}
