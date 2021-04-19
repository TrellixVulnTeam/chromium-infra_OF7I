// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package manifestutil

import (
	"fmt"
	"strings"

	"infra/cros/internal/repo"
)

type MissingProjectsError struct {
	Err             error
	MissingProjects string
}

func (e MissingProjectsError) Error() string {
	return e.Err.Error()
}

// PinManifestFromManifest pins all projects in a manifest to the revisions listed
// for the same projects in a reference manifest.
// If a project does not exist in the reference manifest, it will not be changed
// and an error will be returned. Path is used as the identifier for projects.
func PinManifestFromManifest(manifest, reference *repo.Manifest) error {
	missingProjects := []string{}

	for i := range manifest.Projects {
		if refProject, err := reference.GetProjectByPath(manifest.Projects[i].Path); err != nil {
			missingProjects = append(missingProjects, manifest.Projects[i].Path)
		} else {
			manifest.Projects[i].Revision = refProject.Revision
		}
	}
	if len(missingProjects) > 0 {
		return MissingProjectsError{
			Err:             fmt.Errorf("reference manifest missing projects"),
			MissingProjects: strings.Join(missingProjects, ","),
		}
	}
	return nil
}
