// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package manifestutil

import (
	"context"
	"fmt"
	"log"
	"strings"

	"infra/cros/internal/gerrit"
	"infra/cros/internal/repo"

	bbproto "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
)

const (
	// Name of the root XML file to seek in manifest-internal.
	rootXML = "default.xml"
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

func fetchManifestRecursive(ctx context.Context, gerritClient *gerrit.Client, gc *bbproto.GitilesCommit, file string) (map[string]*repo.Manifest, error) {
	return LoadManifestTreeFromGitiles(ctx, gerritClient, gc.Host, gc.Project, gc.Id, file)
}

// GetRepoToRemoteBranchToSourceRootFromGitiles constructs a Gerrit project to path
// mapping by fetching manifest XML files from Gitiles.
func GetRepoToRemoteBranchToSourceRootFromGitiles(ctx context.Context, gerritClient *gerrit.Client, gc *bbproto.GitilesCommit) (map[string]map[string]string, error) {
	manifests, err := LoadManifestTreeFromGitiles(ctx, gerritClient, gc.Host, gc.Project, gc.Id, rootXML)
	if err != nil {
		return nil, err
	}
	repoToSourceRoot := getRepoToRemoteBranchToSourceRootFromLoadedManifests(manifests)
	log.Printf("Found %d repo to source root mappings from manifest files", len(repoToSourceRoot))
	return repoToSourceRoot, nil
}

// GetRepoToRemoteBranchToSourceRootFromFile constructs a Gerrit project to path
// mapping by reading manifests from the specified root manifest file.
func GetRepoToRemoteBranchToSourceRootFromFile(file string) (map[string]map[string]string, error) {
	manifests, err := LoadManifestTreeFromFile(file)
	if err != nil {
		return nil, errors.Annotate(err, "failed to load local manifest %s", file).Err()
	}
	repoToSourceRoot := getRepoToRemoteBranchToSourceRootFromLoadedManifests(manifests)
	log.Printf("Found %d repo to source root mappings from manifest files", len(repoToSourceRoot))
	return repoToSourceRoot, nil
}

func getRepoToRemoteBranchToSourceRootFromLoadedManifests(manifests map[string]*repo.Manifest) map[string]map[string]string {
	repoToSourceRoot := make(map[string]map[string]string)
	for _, m := range manifests {
		for _, p := range m.Projects {
			if _, found := repoToSourceRoot[p.Name]; !found {
				repoToSourceRoot[p.Name] = make(map[string]string)
			}
			branch := p.Upstream
			if branch == "" {
				branch = "refs/heads/master"
			}
			if !strings.HasPrefix(branch, "refs/heads/") {
				branch = "refs/heads/" + branch
			}

			if oldPath, found := repoToSourceRoot[p.Name][branch]; found {
				log.Printf("Source root for (%s, %s) is currently %s, overwriting with %s", p.Name, branch, oldPath, p.Path)
			}

			repoToSourceRoot[p.Name][branch] = p.Path
		}
	}
	return repoToSourceRoot
}
