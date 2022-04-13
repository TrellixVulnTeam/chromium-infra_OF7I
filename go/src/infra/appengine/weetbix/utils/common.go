// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package utils contains various helper functions.
package utils

import "strings"

// SplitRealm splits the Realm into the LUCI project name and the (sub)Realm.
// Returns empty strings if the provided Realm doesn't have a valid format.
func SplitRealm(realm string) (proj string, subRealm string) {
	parts := strings.SplitN(realm, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
