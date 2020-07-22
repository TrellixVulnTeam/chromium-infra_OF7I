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
	FilterConditionSeparator string = ":"
	Lab                      string = "lab"
	ATL                      string = "atl"
	ACS                      string = "acs"
	Browser                  string = "browser"
	CrOS                     string = "cros"
	ATLLab                   string = "atl-lab"
	ACSLab                   string = "acs-lab"
	BrowserLab               string = "browser-lab"
	CrOSLab                  string = "cros-lab"
)

// Key is a type for use in adding values to context. It is not recommended to use plain string as key.
type Key string

var validLabs = []string{ATLLab, ACSLab, BrowserLab, CrOSLab}

// GetBrowserLabName return a resource name with browser lab prefix and a given user-specified raw name.
func GetBrowserLabName(raw string) string {
	return fmt.Sprintf("%s%s%s", BrowserLab, FilterConditionSeparator, raw)
}

// GetATLLabName returns a resource name with atl-lab prefix and a given user-specified raw name.
func GetATLLabName(raw string) string {
	return fmt.Sprintf("%s%s%s", ATLLab, FilterConditionSeparator, raw)
}

// GetCrOSLabName returns a resource name with ChromeOS lab prefix and a given user-specified raw name.
func GetCrOSLabName(raw string) string {
	return fmt.Sprintf("%s%s%s", CrOSLab, FilterConditionSeparator, raw)
}

// GetLabPrefix returns the lab prefix for the given lab filter
func GetLabPrefix(filter string) string {
	switch filter {
	case Lab + FilterConditionSeparator + Browser:
		return BrowserLab
	case Lab + FilterConditionSeparator + ATL:
		return ATLLab
	case Lab + FilterConditionSeparator + ACS:
		return ACSLab
	default:
		return ""
	}
}

// FormatLabFilter returns a lab filter based on user-specified string name
func FormatLabFilter(userFilter string) string {
	return Lab + FilterConditionSeparator + userFilter
}

// IsInBrowserLab check if a given name(resource or lab name) indicates it's in browser lab.
func IsInBrowserLab(name string) bool {
	// check if it has a browser lab prefix
	s := strings.Split(name, FilterConditionSeparator)
	if len(s) >= 2 && s[0] == BrowserLab {
		return true
	}

	// check the actual lab name
	switch name {
	case "LAB_CHROME_ATLANTA",
		"LAB_DATACENTER_ATL97",
		"LAB_DATACENTER_IAD97",
		"LAB_DATACENTER_MTV96",
		"LAB_DATACENTER_MTV97",
		"LAB_DATACENTER_FUCHSIA":
		return true
	default:
		return false
	}
}

// GetIPName returns a formatted IP name
func GetIPName(vlanName, ipv4Str string) string {
	return fmt.Sprintf("%s/%s", vlanName, ipv4Str)
}

// IsValidFilter checks if a filter is valid
func IsValidFilter(filter string) bool {
	for _, lab := range validLabs {
		if strings.Split(lab, "-")[0] == filter {
			return true
		}
	}
	return false
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
