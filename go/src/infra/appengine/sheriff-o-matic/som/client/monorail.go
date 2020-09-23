// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package client

import (
	"fmt"
	"regexp"

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

var monorailTypeFieldMap = map[string]map[string]string{
	"sheriff-o-matic": {
		"chromium": "projects/chromium/fieldDefs/10",
	},
	"sheriff-o-matic-staging": {
		"chromium": "projects/chromium/fieldDefs/10",
	},
}

// GetMonorailPriorityField get the fieldName for priority.
// TODO (nqmtuan): Put this in admin config.
func GetMonorailPriorityField(c context.Context, projectID string) (string, error) {
	return getFieldValue(c, projectID, monorailPriorityFieldMap)
}

// GetMonorailTypeField get the fieldName for type (e.g. Bug, Feature...).
// TODO (nqmtuan): Put this in admin config.
func GetMonorailTypeField(c context.Context, projectID string) (string, error) {
	return getFieldValue(c, projectID, monorailTypeFieldMap)
}

func getFieldValue(c context.Context, projectID string, fieldMap map[string]map[string]string) (string, error) {
	appID := info.AppID(c)
	val, ok := fieldMap[appID][projectID]
	if !ok {
		return "", fmt.Errorf("Invalid ProjectID %q", projectID)
	}
	return val, nil
}

// GetMonorailProjectResourceName generates Monorail project resource from projectID
func GetMonorailProjectResourceName(projectID string) string {
	return "projects/" + projectID
}

// GetMonorailIssueResourceName generates Monorail issue resource from projectID
// and bugID
func GetMonorailIssueResourceName(projectID string, bugID string) string {
	return fmt.Sprintf("projects/%s/issues/%s", projectID, bugID)
}

// ParseMonorailIssueName gets projectID, bugID from issue resource name
func ParseMonorailIssueName(issueName string) (string, string, error) {
	rgx := regexp.MustCompile("projects/(.+)/issues/(\\d+)")
	rs := rgx.FindStringSubmatch(issueName)
	if len(rs) != 3 {
		return "", "", fmt.Errorf("Invalid resource %q", issueName)
	}
	return rs[1], rs[2], nil
}
