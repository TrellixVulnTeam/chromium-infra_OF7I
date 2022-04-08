// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package utils contains various helper functions.
package utils

import (
	"regexp"
)

// realmProjectRe extracts the LUCI project name from a LUCI Realm.
var realmProjectRe = regexp.MustCompile(`^([a-z0-9\-_]{1,40}):.+$`)

// ProjectFromRealm extracts the LUCI project name from a LUCI Realm.
// Returns an empty string if the provided Realm doesn't have a valid format.
func ProjectFromRealm(realm string) string {
	match := realmProjectRe.FindStringSubmatch(realm)
	if match != nil {
		return match[1]
	}
	return ""
}
