// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"fmt"
	"strconv"
	"strings"
)

// message for filtering
const (
	ColonSeparator string = ":"
	Lab            string = "lab"
	ATL            string = "atl"
	ACS            string = "acs"
	Browser        string = "browser"
	CrOS           string = "cros"
	// https://cloud.google.com/datastore/docs/concepts/limits
	OperationPageSize int = 500
)

// Key is a type for use in adding values to context. It is not recommended to use plain string as key.
type Key string

// GetBrowserLabName return a resource name with browser prefix and a given user-specified raw name.
func GetBrowserLabName(raw string) string {
	return fmt.Sprintf("%s%s%s", Browser, ColonSeparator, raw)
}

// GetATLLabName returns a resource name with atl prefix and a given user-specified raw name.
func GetATLLabName(raw string) string {
	return fmt.Sprintf("%s%s%s", ATL, ColonSeparator, raw)
}

// GetCrOSLabName returns a resource name with ChromeOS prefix and a given user-specified raw name.
func GetCrOSLabName(raw string) string {
	return fmt.Sprintf("%s%s%s", CrOS, ColonSeparator, raw)
}

// IsInBrowserZone check if a given name(resource or zone name) indicates it's in browser zone.
func IsInBrowserZone(name string) bool {
	// check if it has a browser zone prefix
	s := strings.Split(name, ColonSeparator)
	if len(s) >= 2 && s[0] == Browser {
		return true
	}

	// check the actual zone name
	switch name {
	case "ZONE_ATLANTA",
		"ZONE_ATL97",
		"ZONE_IAD97",
		"ZONE_MTV96",
		"ZONE_MTV97",
		"ZONE_FUCHSIA":
		return true
	default:
		return false
	}
}

// GetIPName returns a formatted IP name
func GetIPName(vlanName, ipv4Str string) string {
	return fmt.Sprintf("%s/%s", vlanName, ipv4Str)
}

// Min returns the smaller integer of the two inputs.
func Min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

// Int64ToStr converts the int64 to string
func Int64ToStr(v int64) string {
	return strconv.Itoa(int(v))
}

// Int32ToStr converts the int32 to string
func Int32ToStr(v int32) string {
	return strconv.Itoa(int(v))
}

// RemoveStringEntry removes string entry from the string slice
func RemoveStringEntry(slice []string, entry string) []string {
	for i, s := range slice {
		if s == entry {
			slice[i] = slice[len(slice)-1]
			slice = slice[:len(slice)-1]
			break
		}
	}
	return slice
}

// GetNewNicNameForRenameMachine returns new nic name for new machine name
func GetNewNicNameForRenameMachine(oldNicName, oldMachineName, newMachineName string) string {
	if strings.HasPrefix(oldNicName, oldMachineName+":") {
		return newMachineName + strings.TrimPrefix(oldNicName, oldMachineName)
	}
	return oldNicName
}
