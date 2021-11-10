// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"

	"infra/tools/migrator/internal/migratorpb"
)

// Tweaks represents loaded `tweaks` section of the migrator config file.
type Tweaks struct {
	entries []tweaksEntry
}

type tweaksEntry struct {
	filter Filter
	pb     *migratorpb.Config_ProjectTweaks
}

// ProjectTweaks is config tweaks to apply to a particular LUCI project.
type ProjectTweaks struct {
	Reviewers []string // reviewer emails to send CLs to
	CC        []string // emails to CC CLs to
}

// LoadTweaks parses `tweaks` section of the migrator config file.
func LoadTweaks(projectDir ProjectDir) (*Tweaks, error) {
	cfg, err := projectDir.LoadConfigFile()
	if err != nil {
		return nil, err
	}

	entries := make([]tweaksEntry, 0, len(cfg.Tweaks))
	for _, pb := range cfg.Tweaks {
		filter, err := NewFilter(pb.ProjectsRe)
		if err != nil {
			return nil, errors.Annotate(err, "when loading tweaks").Err()
		}
		entries = append(entries, tweaksEntry{
			filter: filter,
			pb:     pb,
		})
	}

	return &Tweaks{entries: entries}, nil
}

// ProjectTweaks returns config tweaks to apply to a particular LUCI project.
func (t *Tweaks) ProjectTweaks(projectID string) *ProjectTweaks {
	r := stringset.New(0)
	cc := stringset.New(0)

	for _, entry := range t.entries {
		if entry.filter(projectID) {
			r.AddAll(entry.pb.Reviewer)
			cc.AddAll(entry.pb.Cc)
		}
	}

	return &ProjectTweaks{
		Reviewers: r.ToSortedSlice(),
		CC:        cc.ToSortedSlice(),
	}
}
