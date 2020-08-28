// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"fmt"
)

// JoinFilters joins several filters together
func JoinFilters(old []string, filters ...string) []string {
	if len(old) == 0 {
		return filters
	}
	if len(filters) == 0 {
		return old
	}
	res := make([]string, 0, len(old)*len(filters))
	for _, o := range old {
		for _, f := range filters {
			res = append(res, fmt.Sprintf("%s & %s", o, f))
		}
	}
	return res
}

// PrefixFilters returns a group of filter strings with prefix
func PrefixFilters(prefix string, filters []string) []string {
	if len(filters) == 0 {
		return nil
	}
	res := make([]string, len(filters))
	for i, f := range filters {
		res[i] = fmt.Sprintf("%s=%s", prefix, f)
	}
	return res
}
