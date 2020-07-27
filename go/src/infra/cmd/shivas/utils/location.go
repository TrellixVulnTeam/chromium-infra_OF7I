// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"regexp"

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
