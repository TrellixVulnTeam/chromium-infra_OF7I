// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package repo contains functions for interacting with manifests and the
// repo tool.
package repo

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"regexp"

	"go.chromium.org/luci/common/errors"
)

var (
	// Name of the root XML file to seek in manifest-internal.
	rootXML = "default.xml"
)

// Manifest is a top-level Repo definition file.
type Manifest struct {
	XMLName  xml.Name  `xml:"manifest"`
	Revision string    `xml:"revision,attr,omitempty"`
	Includes []Include `xml:"include"`
	Remotes  []Remote  `xml:"remote"`
	Default  Default   `xml:"default"`
	Notice   string    `xml:"notice,omitempty"`
	Projects []Project `xml:"project"`
	// The repo-hooks element references a project. The repo tool will fail if
	// that project has not yet been defined, so we need to put repo-hooks
	// after all projects.
	RepoHooks []RepoHooks `xml:"repo-hooks"`
}

// Project is an element of a manifest containing a Gerrit project to source path definition.
type Project struct {
	Path        string       `xml:"path,attr,omitempty"`
	Name        string       `xml:"name,attr,omitempty"`
	Revision    string       `xml:"revision,attr,omitempty"`
	Upstream    string       `xml:"upstream,attr,omitempty"`
	RemoteName  string       `xml:"remote,attr,omitempty"`
	Annotations []Annotation `xml:"annotation"`
	Groups      string       `xml:"groups,attr,omitempty"`
	SyncC       string       `xml:"sync-c,attr,omitempty"`
	CopyFiles   []CopyFile   `xml:"copyfile"`
}

// Annotation is an element of a manifest annotating the parent element.
type Annotation struct {
	Name  string `xml:"name,attr,omitempty"`
	Value string `xml:"value,attr,omitempty"`
}

// Include is a manifest element that imports another manifest file.
type Include struct {
	Name string `xml:"name,attr,omitempty"`
}

// Remote is a manifest element that lists a remote.
type Remote struct {
	Fetch       string       `xml:"fetch,attr,omitempty"`
	Name        string       `xml:"name,attr,omitempty"`
	Revision    string       `xml:"revision,attr,omitempty"`
	Alias       string       `xml:"alias,attr,omitempty"`
	Annotations []Annotation `xml:"annotation"`
	Review      string       `xml:"review,attr,omitempty"`
}

// Default is a manifest element that lists the default.
type Default struct {
	RemoteName string `xml:"remote,attr,omitempty"`
	Revision   string `xml:"revision,attr,omitempty"`
	SyncJ      string `xml:"sync-j,attr,omitempty"`
}

// CopyFile is a manifest element that lists a copy setting.
type CopyFile struct {
	Dest string `xml:"dest,attr,omitempty"`
	Src  string `xml:"src,attr,omitempty"`
}

// RepoHooks is a manifest element containing various repo hooks.
type RepoHooks struct {
	EnabledList string `xml:"enabled-list,attr,omitempty"`
	InProject   string `xml:"in-project,attr,omitempty"`
}

func (e *Default) AttrMap() map[string]string {
	return map[string]string{
		"remote":   e.RemoteName,
		"revision": e.Revision,
		"sync-j":   e.SyncJ,
	}
}

func (e *Remote) AttrMap() map[string]string {
	return map[string]string{
		"fetch":    e.Fetch,
		"name":     e.Name,
		"revision": e.Revision,
		"alias":    e.Alias,
	}
}

func (e *Project) AttrMap() map[string]string {
	return map[string]string{
		"path":     e.Path,
		"name":     e.Name,
		"revision": e.Revision,
		"upstream": e.Upstream,
		"remote":   e.RemoteName,
		"groups":   e.Groups,
		"sync-c":   e.SyncC,
	}
}

// GitName returns the git name of the remote, which
// is Alias if it is set, and Name otherwise.
func (r *Remote) GitName() string {
	if r.Alias != "" {
		return r.Alias
	}
	return r.Name
}

// GetRemoteByName returns a pointer to the remote with
// the given name/alias in the given manifest.
func (m *Manifest) GetRemoteByName(name string) *Remote {
	for i, remote := range m.Remotes {
		if remote.Name == name {
			return &m.Remotes[i]
		}
	}
	return nil
}

// GetProjectByName returns a pointer to the remote with
// the given path in the given manifest.
func (m *Manifest) GetProjectByName(name string) (*Project, error) {
	for i, project := range m.Projects {
		if project.Name == name {
			return &m.Projects[i], nil
		}
	}
	return nil, fmt.Errorf("project %s does not exist in manifest", name)
}

// GetProjectByPath returns a pointer to the remote with
// the given path in the given manifest.
func (m *Manifest) GetProjectByPath(path string) (*Project, error) {
	for i, project := range m.Projects {
		if project.Path == path {
			return &m.Projects[i], nil
		}
	}
	return nil, fmt.Errorf("project %s does not exist in manifest", path)
}

type projectType string

const (
	singleCheckout projectType = "single"
	multiCheckout  projectType = "multi"
	pinned         projectType = "pinned"
	tot            projectType = "tot"
	drop           projectType = "drop"
)

func (m *Manifest) getProjects(ptype projectType) []*Project {
	projectCount := make(map[string]int)

	for _, project := range m.Projects {
		projectCount[project.Name]++
	}

	projects := []*Project{}
	for i, project := range m.Projects {
		includeProject := false
		projectMode := m.ProjectBranchMode(project)
		if projectMode == Pinned {
			includeProject = ptype == pinned
		} else if projectMode == Tot {
			includeProject = ptype == tot
		} else if projectMode == Drop {
			includeProject = ptype == drop
		} else if projectCount[project.Name] == 1 {
			includeProject = ptype == singleCheckout
		}
		// Restart the if/else if block here because it is possible
		// to have a project with multiple checkouts, some of which
		// are pinned/tot and some of which are not.
		if projectCount[project.Name] > 1 {
			includeProject = includeProject || ptype == multiCheckout
		}
		if includeProject {
			projects = append(projects, &m.Projects[i])
		}
	}
	return projects
}

// GetSingleCheckoutProjects returns all projects in the manifest that have a
// single checkout and are not pinned/tot.
func (m *Manifest) GetSingleCheckoutProjects() []*Project {
	return m.getProjects(singleCheckout)
}

// GetMultiCheckoutProjects returns all projects in the manifest that have a
// multiple checkouts and are not pinned/tot.
func (m *Manifest) GetMultiCheckoutProjects() []*Project {
	return m.getProjects(multiCheckout)
}

// GetPinnedProjects returns all projects in the manifest that are
// pinned.
func (m *Manifest) GetPinnedProjects() []*Project {
	return m.getProjects(pinned)
}

// GetTotProjects returns all projects in the manifest that are
// tot.
func (m *Manifest) GetTotProjects() []*Project {
	return m.getProjects(tot)
}

var (
	gobHost             = "%s.googlesource.com"
	externalGobInstance = "chromium"
	externalGobHost     = fmt.Sprintf(gobHost, externalGobInstance)
	externalGobURL      = fmt.Sprintf("https://%s", externalGobHost)

	internalGobInstance = "chrome-internal"
	internalGobHost     = fmt.Sprintf(gobHost, internalGobInstance)
	internalGobURL      = fmt.Sprintf("https://%s", internalGobHost)

	aospGobInstance = "android"
	aospGobHost     = fmt.Sprintf(gobHost, aospGobInstance)
	aospGobURL      = fmt.Sprintf("https://%s", aospGobHost)

	weaveGobInstance = "weave"
	weaveGobHost     = fmt.Sprintf(gobHost, weaveGobInstance)
	weaveGobURL      = fmt.Sprintf("https://%s", weaveGobHost)

	externalRemote = "cros"
	internalRemote = "cros-internal"

	crosRemotes = map[string]string{
		externalRemote: externalGobURL,
		internalRemote: internalGobURL,
		"aosp":         aospGobURL,
		"weave":        weaveGobURL,
	}

	// Mapping 'remote name' -> regexp that matches names of repositories on
	// that remote that can be branched when creating CrOS branch.
	// Branching script will actually create a new git ref when branching
	// these projects. It won't attempt to create a git ref for other projects
	// that may be mentioned in a manifest. If a remote is missing from this
	// dictionary, all projects on that remote are considered to not be
	// branchable.
	branchableProjects = map[string]*regexp.Regexp{
		externalRemote: regexp.MustCompile("(chromiumos|aosp)/(.+)"),
		internalRemote: regexp.MustCompile("chromeos/(.+)"),
	}

	manifestAttrBranchingCreate = "create"
	manifestAttrBranchingPin    = "pin"
	manifestAttrBranchingToT    = "tot"
	manifestAttrBranchingDrop   = "drop"
)

// BranchMode is a particular branching mode (Pinned, ToT, Create).
type BranchMode string

const (
	// UnspecifiedMode is an unspecified branch mode.
	UnspecifiedMode BranchMode = "unspecified"
	// Pinned branch mode.
	Pinned BranchMode = "pinned"
	// Tot branch mode.
	Tot BranchMode = "tot"
	// Create branch mode.
	Create BranchMode = "create"
	// Drop branch mode.
	Drop BranchMode = "drop"
)

// ProjectBranchMode returns the branch mode (create, pinned, tot, drop) of a project.
func (m *Manifest) ProjectBranchMode(project Project) BranchMode {
	// Anotation is set.
	explicitMode, _ := project.GetAnnotation("branch-mode")
	if explicitMode != "" {
		switch explicitMode {
		case manifestAttrBranchingCreate:
			return Create
		case manifestAttrBranchingPin:
			return Pinned
		case manifestAttrBranchingToT:
			return Tot
		case manifestAttrBranchingDrop:
			return Drop
		default:
			return UnspecifiedMode
		}
	}

	// Othwerise, peek at remote.
	remote := m.GetRemoteByName(project.RemoteName)
	if remote == nil {
		return UnspecifiedMode
	}
	remoteName := remote.GitName()
	_, inCrosRemote := crosRemotes[remoteName]
	projectRegexp, inBranchableProjects := branchableProjects[remoteName]

	if inCrosRemote && inBranchableProjects && projectRegexp.MatchString(project.Name) {
		return Create
	}
	return Pinned
}

// GetAnnotation returns the value of the annotation with the
// given name, if it exists. It also returns a bool indicating
// whether or not the annotation exists.
func (p *Project) GetAnnotation(name string) (string, bool) {
	for _, annotation := range p.Annotations {
		if annotation.Name == name {
			return annotation.Value, true
		}
	}
	return "", false
}

// GetAnnotation returns the value of the annotation with the
// given name, if it exists. It also returns a bool indicating
// whether or not the annotation exists.
func (r *Remote) GetAnnotation(name string) (string, bool) {
	for _, annotation := range r.Annotations {
		if annotation.Name == name {
			return annotation.Value, true
		}
	}
	return "", false
}

// ResolveImplicitLinks explicitly sets remote/revision information
// for each project in the manifest.
func (m *Manifest) ResolveImplicitLinks() {
	for i, project := range m.Projects {
		// Set default remote on projects without an explicit remote
		if project.RemoteName == "" {
			project.RemoteName = m.Default.RemoteName
		}
		// Set default revision on projects without an explicit revision
		if project.Revision == "" {
			remote := m.GetRemoteByName(project.RemoteName)
			if remote == nil || remote.Revision == "" {
				project.Revision = m.Default.Revision
			} else {
				project.Revision = remote.Revision
			}
		}
		// Path defaults to name.
		if project.Path == "" {
			project.Path = project.Name
		}
		m.Projects[i] = project
	}
}

// GetUniqueProject returns the unique project with the given name
// (nil if the project DNE). It returns an error if multiple projects with the
// given name exist.
func (m *Manifest) GetUniqueProject(name string) (Project, error) {
	var project Project
	matchingProjects := 0
	for _, p := range m.Projects {
		if p.Name == name {
			matchingProjects++
			if matchingProjects > 1 {
				return Project{}, fmt.Errorf("multiple projects named %s", name)
			}
			project = p
		}
	}
	if matchingProjects == 0 {
		return Project{}, fmt.Errorf("no project named %s", name)
	}
	return project, nil
}

// ParseManifest parses a manifest from the given byte array.
func ParseManifest(contents []byte) (*Manifest, error) {
	manifest := &Manifest{}
	if err := xml.Unmarshal(contents, manifest); err != nil {
		return nil, errors.Annotate(err, "failed to unmarshal data").Err()
	}
	return manifest, nil
}

// ToBytes marshals the manifest into a raw byte array.
func (m *Manifest) ToBytes() ([]byte, error) {
	data, err := xml.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, errors.Annotate(err, "failed to write manifest").Err()
	}
	return data, nil
}

// WriteToBytes returns the manifest as a byte array in its final output format.
func (m *Manifest) WriteToBytes() ([]byte, error) {
	data, err := m.ToBytes()
	if err != nil {
		return nil, err
	}
	return []byte(xml.Header + string(data)), nil
}

// Write writes the manifest to the given path.
func (m *Manifest) Write(path string) error {
	data, err := m.WriteToBytes()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, data, 0644)
	if err != nil {
		return errors.Annotate(err, "failed to write manifest").Err()
	}
	return nil
}

func projectInArr(project Project, projects []Project) bool {
	for _, proj := range projects {
		// Path is a unniue identifier for projects.
		if project.Path == proj.Path {
			return true
		}
	}
	return false
}

func remoteInArr(remote Remote, remotes []Remote) bool {
	for _, rem := range remotes {
		if remote.Name == rem.Name {
			return true
		}
	}
	return false
}

// MergeManifests will merge the given manifests based on includes, taking
// manifests[path] to be the top-level manifest.
// manifests maps manifest filenames to the Manifest structs themselves.
// This basically re-implements `repo manifest` but is necessary because we can't run
// `repo manifest` on a singular git repository.
func MergeManifests(root string, manifests *map[string]*Manifest) (*Manifest, error) {
	baseManifest, ok := (*manifests)[root]
	if !ok {
		return nil, fmt.Errorf("manifest %s does not exist", root)
	}

	// Merge each included manifest. Then merge into baseManifest.
	for _, includedManifest := range baseManifest.Includes {
		mergedManifest, err := MergeManifests(includedManifest.Name, manifests)
		if err != nil {
			return nil, err
		}
		// Merge projects.
		for _, project := range mergedManifest.Projects {
			if !projectInArr(project, baseManifest.Projects) {
				baseManifest.Projects = append(baseManifest.Projects, project)
			}
		}
		// Merge remotes.
		for _, remote := range mergedManifest.Remotes {
			if !remoteInArr(remote, baseManifest.Remotes) {
				baseManifest.Remotes = append(baseManifest.Remotes, remote)
			}
		}
		// Keep the base manifest's default.
	}

	// Clear includes.
	baseManifest.Includes = []Include{}
	baseManifest.ResolveImplicitLinks()
	return baseManifest, nil
}
