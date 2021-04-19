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
