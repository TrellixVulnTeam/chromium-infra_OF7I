// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package client

import (
	"fmt"

	"go.chromium.org/luci/gae/service/info"
	"golang.org/x/net/context"
)

// monorailPriorityFieldMap records the resource name of priority field
// in different projects and environment.
var monorailPriorityFieldMap = map[string]map[string]string{
	"sheriff-o-matic": {
		"chromium": "projects/chromium/fieldDefs/11",
		"fuchsia":  "projects/fuchsia/fieldDefs/168",
	},
	"sheriff-o-matic-staging": {
		"chromium": "projects/chromium/fieldDefs/11",
		"fuchsia":  "projects/fuchsia/fieldDefs/246",
	},
}

// GetMonorailPriorityField get the fieldName for priority.
// TODO (nqmtuan): Put this in admin config.
func GetMonorailPriorityField(c context.Context, projectID string) (string, error) {
	appID := info.AppID(c)
	val, ok := monorailPriorityFieldMap[appID][projectID]
	if !ok {
		return "", fmt.Errorf("Invalid ProjectID %q", projectID)
	}
	return val, nil
}
