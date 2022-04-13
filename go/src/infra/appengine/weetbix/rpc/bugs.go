// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"fmt"

	"infra/appengine/weetbix/internal/bugs"
	configpb "infra/appengine/weetbix/internal/config/proto"
	pb "infra/appengine/weetbix/proto/v1"
)

func createAssociatedBugPB(b bugs.BugID, cfg *configpb.ProjectConfig) *pb.AssociatedBug {
	// Fallback bug name and URL.
	linkText := fmt.Sprintf("%s/%s", b.System, b.ID)
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
				linkText = fmt.Sprintf("%s/%s", cfg.Monorail.DisplayPrefix, id)
			} else {
				linkText = id
			}
		}
		if cfg.Monorail.MonorailHostname != "" {
			url = fmt.Sprintf("https://%s/p/%s/issues/detail?id=%s", cfg.Monorail.MonorailHostname, project, id)
		}
	case bugs.BuganizerSystem:
		linkText = fmt.Sprintf("b/%s", b.ID)
		url = fmt.Sprintf("https://issuetracker.google.com/issues/%s", b.ID)
	default:
		// Fallback.
	}
	return &pb.AssociatedBug{
		System:   b.System,
		Id:       b.ID,
		LinkText: linkText,
		Url:      url,
	}
}
