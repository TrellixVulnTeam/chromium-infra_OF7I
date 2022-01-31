// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"regexp"
)

// ProjectRePattern is the regular expression pattern that matches
// validly formed LUCI Project names.
// From https://source.chromium.org/chromium/infra/infra/+/main:luci/appengine/components/components/config/common.py?q=PROJECT_ID_PATTERN
const ProjectRePattern = `[a-z0-9\-]{1,40}`

// ProjectRe matches validly formed LUCI Project names.
var ProjectRe = regexp.MustCompile(`^` + ProjectRePattern + `$`)
