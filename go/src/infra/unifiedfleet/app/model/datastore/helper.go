// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package datastore

// message for filtering
const (
	FilterConditionSeparator string = ":"
	Lab                      string = "lab"
	ATL                      string = "atl"
	ACS                      string = "acs"
	Browser                  string = "browser"
	ATLLab                   string = "atl-lab:"
	ACSLab                   string = "acs-lab:"
	BrowserLab               string = "browser-lab:"
)

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
