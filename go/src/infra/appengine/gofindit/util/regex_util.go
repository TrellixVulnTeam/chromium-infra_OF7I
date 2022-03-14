// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"fmt"
	"regexp"
)

// MatchedNamedGroup matches a string against a regex with named group
// and returns a mapping between the named group and the matches
func MatchedNamedGroup(r *regexp.Regexp, s string) (map[string]string, error) {
	names := r.SubexpNames()
	matches := r.FindStringSubmatch(s)
	result := make(map[string]string)
	if matches != nil {
		for i, name := range names {
			if name != "" {
				result[name] = matches[i]
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("Could not find matches")
}
