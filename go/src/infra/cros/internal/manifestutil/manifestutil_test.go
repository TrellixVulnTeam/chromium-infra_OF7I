// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// +build linux

package manifestutil

import (
	"testing"

	"infra/cros/internal/assert"
	"infra/cros/internal/repo"
)

func TestPinManifestFromManifest(t *testing.T) {
	manifest := &repo.Manifest{
		Projects: []repo.Project{
			{
				Path:     "foo/",
				Name:     "foo",
				Revision: "ref-foo",
			},
			{
				Path:     "bar/",
				Name:     "bar",
				Revision: "ref-bar",
			},
		},
	}
	reference := &repo.Manifest{
		Projects: []repo.Project{
			{
				Path:     "foo/",
				Name:     "foo",
				Revision: "ref-foo-new",
			},
		},
	}

	err := PinManifestFromManifest(manifest, reference)
	assert.ErrorContains(t, err, "missing")
	assert.StringsEqual(t, manifest.Projects[0].Revision, "ref-foo-new")
	assert.StringsEqual(t, manifest.Projects[1].Revision, "ref-bar")
}

func TestGetRepoToRemoteBranchToSourceRootFromFile_success(t *testing.T) {
	m, err := GetRepoToRemoteBranchToSourceRootFromFile("test_data/foo.xml")
	if err != nil {
		t.Error(err)
	}
	if len(m) != 4 {
		t.Errorf("expected %d project mappings, found %d", 4, len(m))
	}
	// Make sure that a sample project is present.
	if m["baz"]["refs/heads/master"] != "baz/" {
		t.Errorf("expected to find a mapping for baz. Got mappings: %v", m)
	}
}

func TestGetRepoToRemoteBranchToSourceRootFromFile_duplicate(t *testing.T) {
	m, err := GetRepoToRemoteBranchToSourceRootFromFile("test_data/duplicate.xml")
	if err != nil {
		t.Error(err)
	}
	if len(m) != 1 {
		t.Errorf("expected %d project mappings, found %d", 1, len(m))
	}
	// The last mapping for a given name and branch should take precedent.
	if m["foo"]["refs/heads/master"] != "buz/" {
		t.Errorf("expected to find a mapping for buz. Got mappings: %v", m)
	}
}
