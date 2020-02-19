// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"regexp"
)

var locations = []*regexp.Regexp{
	regexp.MustCompile(`ROW[\d]*-RACK[\d]*-HOST[\d]*`),
	regexp.MustCompile(`chromeos[\d]*-row[\d]*-rack[\d]*-host[\d]*`),
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
