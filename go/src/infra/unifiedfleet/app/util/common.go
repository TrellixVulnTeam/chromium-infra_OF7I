// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
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

	// Version Key format of inventory
	TimestampBasedVersionKeyFormat string = "2006-01-02 15:04:05.000 UTC"
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
		"ZONE_FUCHSIA",
		"ZONE_BROWSER_GOOGLER_DESK",
		"ZONE_SFO36_BROWSER":
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

// ContainsAnyStrings returns true if at least one of the inputs is in list.
func ContainsAnyStrings(list []string, inputs ...string) bool {
	for _, a := range list {
		for _, b := range inputs {
			if b == a {
				return true
			}
		}
	}
	return false
}

// ProtoEqual compares the given protos i, j and returns true if Type(i) == Type(j) and one of the following holds
// 1. i == j == nil
// 2. For each exported field(proto message field) x in Type(i), i.Equal(j) or i.x == j.y.
// returns false otherwise.
func ProtoEqual(i, j proto.Message) bool {
	// Check if both the inputs are of same type. Cannot compare dissimilar protos.
	if proto.MessageReflect(i).Descriptor().FullName() != proto.MessageReflect(j).Descriptor().FullName() {
		return false
	}

	if i == nil || j == nil {
		return i == j
	}

	// Create a ignore unexported paths filter for comparision.
	opt := cmp.FilterPath(func(p cmp.Path) bool {
		// Filters the unexported paths from the given protos. Returns true if path is unexported.

		// Get the last path Ex: MachineLSE.ChromeosMachineLse.DeviceLse.Dut.Peripherals.Servo -> Servo
		lPath := p.Index(-1)
		// Check if its a struct field.
		sf, ok := lPath.(cmp.StructField)
		if !ok {
			// path is a pointer to struct. Compare the struct.
			return false
		}
		// Decode the first rune in the variable name
		r, _ := utf8.DecodeRuneInString(sf.Name())
		// Exported field names start with upper case alphabet. Check if it's Upper case.
		return !unicode.IsUpper(r)
	}, cmp.Ignore()) // Ignore the unexported paths.

	// Compare the proto message.
	return cmp.Equal(i, j, opt)
}

var (
	// satlabRegex regular expression to get the hive value from a DUT hostname.
	satlabRegex = regexp.MustCompile(`^satlab-[^-]+`)

	// gtransitRegex regular expression to identify a gTransit DUT hostname.
	gtransitRegex = regexp.MustCompile(`^cros-mtv1950-144-rack[\d]+`)

	// Lab SFO36_OS will start with the DUT naming of 'chromeos8-...'.
	sfo36OSRegex = regexp.MustCompile(`^chromeos8-`)
)

// gtransitHive hive value for a gTransit DUT.
const gtransitHive string = "cros-mtv1950-144"

// GetHiveForDut returns the hive value for a DUT.
//
// hive value is derived from the DUT hostname.
func GetHiveForDut(hostname string) string {
	// gTransit DUTs.
	if gtransitRegex.MatchString(hostname) {
		return gtransitHive
	}
	if sfo36OSRegex.MatchString(hostname) {
		// 'e' is site site letter assigned to SFO36.
		return "e"
	}
	// Satlab DUTs.
	if satlabRegex.MatchString(hostname) {
		return satlabRegex.FindString(hostname)
	}
	// Main lab DUTs.
	return ""
}

// AppendUniqueStrings returns a slice with unique elements such that all the des elements are appended to src.
func AppendUniqueStrings(des []string, src ...string) []string {
	sliceWithDups := sort.StringSlice(append(des, src...))
	sliceWithDups.Sort()
	var uniqueSlice []string
	var prev string
	// Filter the sorted array for duplicates linearly.
	for _, elem := range sliceWithDups {
		if elem != prev {
			uniqueSlice = append(uniqueSlice, elem)
		}
		prev = elem
	}
	return uniqueSlice
}
