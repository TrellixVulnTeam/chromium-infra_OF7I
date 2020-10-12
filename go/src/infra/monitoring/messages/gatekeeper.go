// Copyright 2015 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// These structs are used for parsing gatekeeper.json files.

package messages

import (
	"net/url"
	"strings"
)

// BuilderGroupLocation is the location of a builder group.
// Currently it's just a URL.
type BuilderGroupLocation struct {
	url.URL
}

// Name is the name of the builder group; chromium, chromium.linux, etc.
func (m *BuilderGroupLocation) Name() string {
	parts := strings.Split(m.Path, "/")
	return parts[len(parts)-1]
}
