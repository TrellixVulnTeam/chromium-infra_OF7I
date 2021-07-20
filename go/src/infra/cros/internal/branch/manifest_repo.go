// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package branch

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"infra/cros/internal/git"
	"infra/cros/internal/manifestutil"
	"infra/cros/internal/repo"

	"go.chromium.org/luci/common/errors"
)

type ManifestRepo struct {
	ProjectCheckout string
	Project         repo.Project
}

const (
	defaultManifest  = "default.xml"
	officialManifest = "official.xml"
)

const (
	attrRegexpTemplate = "%s=\"[^\"]*\""
	tagRegexpTempate   = "<%s[^(<>)]*>"
)

var loadManifestFromFileRaw = manifestutil.LoadManifestFromFileRaw
var loadManifestTree = manifestutil.LoadManifestTreeFromFile

func (c *Client) gitRevision(m *ManifestRepo, project repo.Project) (string, error) {
	if git.IsSHA(project.Revision) {
		return project.Revision, nil
	}

	remoteURL, err := c.ProjectFetchURL(project.Path)
	if err != nil {
		return "", err
	}

	// Doesn't need to be in an actual git repo.
	output, err := git.RunGit("", []string{"ls-remote", remoteURL, project.Revision})
	if err != nil {
		return "", errors.Annotate(err, "failed to read remote branches for %s", remoteURL).Err()
	}
	if strings.TrimSpace(output.Stdout) == "" {
		return "", fmt.Errorf("no Ref for %s in project %s", project.Revision, project.Path)
	}
	return strings.Fields(output.Stdout)[0], nil
}

func delAttr(tag, attr string) string {
	// Regex for finding attribute. Include leading whitespace.
	attrRegex := regexp.MustCompile(fmt.Sprintf(`\s*`+attrRegexpTemplate, attr))
	return attrRegex.ReplaceAllString(tag, ``)
}

func setAttr(tag, attr, value string) string {
	// Regex for finding attribute.
	attrRegex := regexp.MustCompile(fmt.Sprintf(attrRegexpTemplate, attr))
	// Attribute with new value.
	newAttr := fmt.Sprintf(`%s="%s"`, attr, value)

	// Attribute with current value.
	currAttr := attrRegex.FindString(tag)
	if currAttr != "" { // Attr exists, replace value.
		return attrRegex.ReplaceAllString(tag, newAttr)
	}
	// Attr does not exist, add attribute to end of [start] tag.
	endRegex := regexp.MustCompile(`(\s*/?>)`)
	return endRegex.ReplaceAllString(tag, " "+newAttr+"$1")
}

func setRevisionAttr(tag, revision string) string {
	return setAttr(tag, "revision", revision)
}

// Given a repo.Project struct, find the corresponding start tag in
// a raw XML file. Empty string indicates no match.
func findProjectTag(project *repo.Project, rawManifest string) string {
	projectRegexp := regexp.MustCompile(fmt.Sprintf(tagRegexpTempate, "project"))
	for _, tag := range projectRegexp.FindAllString(rawManifest, -1) {
		p := &repo.Project{}

		// If tag is not a singleton, add empty end tag for unmarshalling purposes.
		var err error
		if tag[len(tag)-2:] != "/>" {
			err = xml.Unmarshal([]byte(tag+"</project>"), p)
		} else {
			err = xml.Unmarshal([]byte(tag), p)
		}
		if err != nil {
			continue
		}

		// Together, Name and Path form a unique identifier.
		// If Path is blank, Name is (or at least ought to be) a unique identifier.
		if project.Name == p.Name && (p.Path == "" || project.Path == p.Path) {
			return tag
		}
	}
	return ""
}

// repairManifest reads the manifest at the given path and repairs it in memory.
// Because humans rarely read branched manifests, this function optimizes for
// code readability and explicitly sets revision on every project in the manifest,
// deleting any defaults.
// branchesByPath maps project paths to branch names.
func (c *Client) repairManifest(m *ManifestRepo, path string, branchesByPath map[string]string) ([]byte, error) {
	manifestData, err := loadManifestFromFileRaw(path)
	if err != nil {
		return nil, errors.Annotate(err, "error loading manifest").Err()
	}

	manifest := repo.Manifest{}
	err = xml.Unmarshal(manifestData, &manifest)
	if err != nil {
		return nil, errors.Annotate(err, "failed to unmarshal manifest").Err()
	}
	manifest.ResolveImplicitLinks()

	// Delete the default revision.
	manifest.Default.Revision = ""

	// Delete remote revisions.
	for i := range manifest.Remotes {
		manifest.Remotes[i].Revision = ""
	}

	// Update all project revisions.
	for i, project := range manifest.Projects {
		// Path defaults to name.
		if project.Path == "" {
			project.Path = project.Name
		}

		workingProject, err := c.WorkingManifest.GetProjectByPath(project.Path)
		if err != nil {
			// We don't really know what to do with a project that doesn't exist in the working manifest,
			// which is our source of truth. Our best bet is to just use what we have in the manifest
			// we're repairing.
			c.LogErr("Warning: project %s does not exist in working manifest. Using it as it exists in %s.", project.Path, path)
			continue
		}

		switch branchMode := c.WorkingManifest.ProjectBranchMode(project); branchMode {
		case repo.Create:
			branchName, inDict := branchesByPath[project.Path]
			if !inDict {
				return nil, fmt.Errorf("project %s is not pinned/tot but not set in branchesByPath", project.Path)
			}
			project.Revision = git.NormalizeRef(branchName)
		case repo.Tot:
			// Correct default would have already been found in ResolveImplicitLinks,
			// so if the revision is blank we have an invalid manifest (no revision + branch_mode ToT).
			// See b/184966242 for context.
			if project.Revision == "" {
				return nil, fmt.Errorf("project %s tracking ToT but has no revision/default revision", project.Path)
			}
		case repo.Pinned:
			revision, err := c.gitRevision(m, *workingProject)
			if err != nil {
				return nil, errors.Annotate(err, "error repairing manifest").Err()
			}
			project.Revision = revision
		default:
			return nil, fmt.Errorf("project %s branch mode unspecifed", project.Path)
		}
		// UpdateManifestElement will write all attributes to the file, so
		// clear the remote if it matches the default.
		if project.RemoteName == manifest.Default.RemoteName {
			project.RemoteName = ""
		}
		// Clear upstream.
		project.Upstream = ""
		manifest.Projects[i] = project
	}

	return manifestutil.UpdateManifestElements(&manifest, manifestData)
}

// listManifests finds all manifests included directly or indirectly by root
// manifests.
func (m *ManifestRepo) listManifests(rootPaths []string) ([]string, error) {
	manifestPaths := make(map[string]bool)

	for _, path := range rootPaths {
		path = filepath.Join(m.ProjectCheckout, path)
		manifestMap, err := loadManifestTree(path)
		if err != nil {
			// It is only correct to continue when a file does not exist,
			// not because of other errors (like invalid XML).
			if strings.Contains(err.Error(), "failed to open") {
				continue
			} else {
				return []string{}, err
			}
		}
		for k := range manifestMap {
			manifestPaths[filepath.Join(filepath.Dir(path), k)] = true
		}
	}
	manifests := []string{}
	for k := range manifestPaths {
		manifests = append(manifests, k)
	}
	return manifests, nil
}

// RepairManifestsOnDisk repairs the revision and upstream attributes of
// manifest elements on disk for the given projects.
func (c *Client) RepairManifestsOnDisk(m *ManifestRepo, branchesByPath map[string]string) error {
	c.LogOut("Repairing manifest project %s", m.Project.Name)
	manifestPaths, err := m.listManifests([]string{defaultManifest, officialManifest})

	if err != nil {
		return errors.Annotate(err, "failed to listManifests").Err()
	}
	for _, manifestPath := range manifestPaths {
		manifest, err := c.repairManifest(m, manifestPath, branchesByPath)
		if err != nil {
			return errors.Annotate(err, "failed to repair manifest %s", manifestPath).Err()
		}
		err = ioutil.WriteFile(manifestPath, manifest, 0644)
		if err != nil {
			return errors.Annotate(err, "failed to write repaired manifest to %s", manifestPath).Err()
		}
	}
	return nil
}
