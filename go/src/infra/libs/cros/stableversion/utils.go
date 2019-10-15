// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package stableversion

import (
	"fmt"
	"regexp"
	"strconv"
)

// findMatchMap takes a regexp and a string and returns a map
// associating the named capture groups in the regexp to their values.
//
// It is intended to be used primarily with regexps such as the one below.
//   regexp.MustCompile(`\AR(?P<release>[0-9]+)-(?P<tip>[0-9]+)\.(?P<branch>[0-9]+)\.(?P<branchbranch>[0-9]+)\z`)
//
// The regexp above only has named capture groups and always matches exactly once because of the \A and \z anchors.
// The intended use case is to use a regexp such as the one above as a blueprint for converting a string into a map
// from strings to strings.
func findMatchMap(r *regexp.Regexp, s string) (map[string]string, error) {
	if r == nil {
		return nil, fmt.Errorf("*regexp cannot be nil")
	}
	out := make(map[string]string)
	matchNames := r.SubexpNames()
	matchValues := r.FindStringSubmatch(s)
	if len(matchNames) != len(matchValues) {
		return nil, fmt.Errorf("mismatch between len(matchNames) (%d) and len(matchValues) (%d)", len(matchNames), len(matchValues))
	}
	for i, name := range matchNames {
		out[name] = matchValues[i]
	}
	return out, nil
}

func parseInt32(s string) (int32, error) {
	i64, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(i64), nil
}

func extractInt32(m map[string]string, k string) (int32, error) {
	if value, ok := m[k]; ok {
		if value == "" {
			return 0, fmt.Errorf("%s cannot be empty", k)
		}
		return parseInt32(value)
	}
	return 0, fmt.Errorf("key %s must be present", k)
}

func extractString(m map[string]string, k string) (string, error) {
	if value, ok := m[k]; ok {
		return value, nil
	}
	return "", fmt.Errorf("key %s must be present", k)
}
