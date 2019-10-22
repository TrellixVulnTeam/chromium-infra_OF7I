// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package buildstatus

import (
	"net/url"
	"regexp"
	"strconv"

	"go.chromium.org/luci/common/errors"
)

var miloPathRX = regexp.MustCompile(
	`/b/(?P<buildID>\d+)`)

// ParseBuildURL obtains buildID from the build url.
func ParseBuildURL(rawURL string) (buildID int64, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return
	}
	m := miloPathRX.FindStringSubmatch(u.Path)
	names := miloPathRX.SubexpNames()
	if len(m) < len(names) || m == nil {
		err = errors.Reason("The path given does not match the expected format. %s", u.Path).Err()
		return
	}
	parts := map[string]string{}
	for i, name := range names {
		if i != 0 {
			parts[name] = m[i]
		}
	}
	buildIDS, hasBuildID := parts["buildID"]
	if !(hasBuildID) {
		err = errors.Reason("The path given does not match the expected format. %s", u.Path).Err()
		return
	}

	buildIDI, err := strconv.ParseInt(buildIDS, 10, 64)
	if err != nil {
		return
	}
	buildID = int64(buildIDI)
	return
}
