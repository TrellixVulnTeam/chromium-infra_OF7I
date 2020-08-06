// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"fmt"
	"regexp"
	"strings"

	"go.chromium.org/luci/common/errors"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsUtil "infra/unifiedfleet/app/util"
)

var locations = []*regexp.Regexp{
	regexp.MustCompile(`ROW[\d]*-RACK[\d]*-HOST[\d]*`),
	regexp.MustCompile(`chromeos[\d]*-row[\d]*-rack[\d]*-host[\d]*`),
	regexp.MustCompile(`chromeos[\d]*-row[\d]*-rack[\d]*-labstation[\d]*`),
	regexp.MustCompile(`chromeos[\d]*-floor`),
	regexp.MustCompile(`chromeos[\d]*-rack[\d]*`),
	regexp.MustCompile(`[\w]*[\d]*-storage[\d]*`),
	regexp.MustCompile(`[\w]*[\d]*-container[\d]*`),
	regexp.MustCompile(`[\w]*[\d]*-desk-[\w]*`),
}

var labFilterRegex = regexp.MustCompile(`lab=[a-zA-Z0-9-,_]*`)

// IsLocation determines if a string describes a ChromeOS lab location
func IsLocation(iput string) bool {
	for _, exp := range locations {
		if exp.MatchString(iput) {
			return true
		}
	}
	return false
}

// StrToUFSLab refers a map between a string to a UFS defined map.
var StrToUFSLab = map[string]string{
	"atl":        "LAB_CHROME_ATLANTA",
	"chromeos1":  "LAB_CHROMEOS_SANTIAM",
	"chromeos4":  "LAB_CHROMEOS_DESTINY",
	"chromeos6":  "LAB_CHROMEOS_PROMETHEUS",
	"chromeos2":  "LAB_CHROMEOS_ATLANTIS",
	"chromeos3":  "LAB_CHROMEOS_LINDAVISTA",
	"chromeos5":  "LAB_CHROMEOS_LINDAVISTA",
	"chromeos7":  "LAB_CHROMEOS_LINDAVISTA",
	"chromeos9":  "LAB_CHROMEOS_LINDAVISTA",
	"chromeos15": "LAB_CHROMEOS_LINDAVISTA",
	"atl97":      "LAB_DATACENTER_ATL97",
	"iad97":      "LAB_DATACENTER_IAD97",
	"mtv96":      "LAB_DATACENTER_MTV96",
	"mtv97":      "LAB_DATACENTER_MTV97",
	"lab01":      "LAB_DATACENTER_FUCHSIA",
	"unknown":    "LAB_UNSPECIFIED",
}

// IsUFSLab checks if a string refers to a valid UFS lab.
func IsUFSLab(lab string) bool {
	_, ok := StrToUFSLab[lab]
	return ok
}

// ValidLabStr returns a valid str list for lab strings.
func ValidLabStr() []string {
	ks := make([]string, 0, len(StrToUFSLab))
	for k := range StrToUFSLab {
		ks = append(ks, k)
	}
	return ks
}

// ToUFSLab converts lab string to a UFS lab enum.
func ToUFSLab(lab string) ufspb.Lab {
	v, ok := StrToUFSLab[lab]
	if !ok {
		return ufspb.Lab_LAB_UNSPECIFIED
	}
	return ufspb.Lab(ufspb.Lab_value[v])
}

// ToUFSRealm returns the realm name based on lab string.
func ToUFSRealm(lab string) string {
	ufsLab := ToUFSLab(lab)
	if ufsUtil.IsInBrowserLab(ufsLab.String()) {
		return ufsUtil.BrowserLabAdminRealm
	}
	if ufsLab == ufspb.Lab_LAB_CHROMEOS_LINDAVISTA {
		return ufsUtil.AcsLabAdminRealm
	}
	return ufsUtil.AtlLabAdminRealm
}

// ReplaceLabNames replace lab names with Ufs lab names in the filter string
//
// TODO(eshwarn) : Repalce regex and string matching with yacc (https://godoc.org/golang.org/x/tools/cmd/goyacc)
func ReplaceLabNames(filter string) (string, error) {
	if filter != "" {
		// Remove all the spaces
		filter = fmt.Sprintf(strings.Replace(filter, " ", "", -1))
		// Find the lab filtering condition
		labfilters := labFilterRegex.FindAllString(filter, -1)
		if len(labfilters) == 0 {
			return filter, nil
		}
		// Aggregate all the lab names
		var labNames []string
		for _, lf := range labfilters {
			keyValue := strings.Split(lf, "=")
			if len(keyValue) < 2 {
				return filter, nil
			}
			labNames = append(labNames, strings.Split(keyValue[1], ",")...)
		}
		if len(labNames) == 0 {
			return filter, nil
		}
		// Repalce all the lab names with UFS lab names matching the enum
		for _, labName := range labNames {
			ufsLabName, ok := StrToUFSLab[labName]
			if !ok {
				errorMsg := fmt.Sprintf("Invalid lab name %s for filtering.\nValid lab filters: [%s]", labName, strings.Join(ValidLabStr(), ", "))
				return filter, errors.New(errorMsg)
			}
			filter = fmt.Sprintf(strings.Replace(filter, labName, ufsLabName, -1))
		}
	}
	return filter, nil
}
