// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handlers

import (
	"fmt"

	"infra/appengine/weetbix/internal/bugs"
	configpb "infra/appengine/weetbix/internal/config/proto"
)

// BugLink provides the details of how to link to a bug in an issue
// tracking system.
type BugLink struct {
	// BugName is the human-readable display name of the bug.
	// E.g. "crbug.com/123456".
	// Output only.
	Name string `json:"name"`
	// BugURL is the link to the bug.
	// E.g. "https://bugs.chromium.org/p/chromium/issues/detail?id=123456".
	// Output only.
	URL string `json:"url"`
}

// createBugLink crates a BugLink from a BugID.
func createBugLink(b bugs.BugID, cfg *configpb.ProjectConfig) *BugLink {
	// Fallback bug name and URL.
	name := fmt.Sprintf("%s/%s", b.System, b.ID)
	url := ""

	switch b.System {
	case bugs.MonorailSystem:
		project, id, err := b.MonorailProjectAndID()
		if err != nil {
			// Fallback to basic name and blank URL.
			break
		}
		if project == cfg.Monorail.Project {
			if cfg.Monorail.DisplayPrefix != "" {
				name = fmt.Sprintf("%s/%s", cfg.Monorail.DisplayPrefix, id)
			} else {
				name = id
			}
		}
		if cfg.Monorail.MonorailHostname != "" {
			url = fmt.Sprintf("https://%s/p/%s/issues/detail?id=%s", cfg.Monorail.MonorailHostname, project, id)
		}
	case bugs.BuganizerSystem:
		name = fmt.Sprintf("b/%s", b.ID)
		url = fmt.Sprintf("https://issuetracker.google.com/issues/%s", b.ID)
	default:
		// Fallback.
	}
	return &BugLink{
		Name: name,
		URL:  url,
	}
}
