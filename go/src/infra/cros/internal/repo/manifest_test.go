// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package repo

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"infra/cros/internal/assert"
)

func ManifestEq(a, b *Manifest) bool {
	if len(a.Projects) != len(b.Projects) {
		return false
	}
	for i := range a.Projects {
		if !reflect.DeepEqual(&a.Projects[i], &b.Projects[i]) {
			return false
		}
	}
	if len(a.Includes) != len(b.Includes) {
		return false
	}
	for i := range a.Includes {
		if a.Includes[i] != b.Includes[i] {
			return false
		}
	}
	return true
}

func ManifestMapEq(expected, actual map[string]*Manifest) error {
	for file := range expected {
		if _, ok := actual[file]; !ok {
			return fmt.Errorf("missing manifest %s", file)
		}
		if !ManifestEq(expected[file], actual[file]) {
			return fmt.Errorf("expected %v, found %v", expected[file], actual[file])
		}
	}
	return nil
}

func TestResolveImplicitLinks(t *testing.T) {
	manifest := &Manifest{
		Default: Default{
			RemoteName: "chromeos",
			Revision:   "123",
		},
		Remotes: []Remote{
			{
				Fetch: "https://chromium.org/remote",
				Name:  "chromium",
				Alias: "chromeos",
			},
			{
				Fetch:    "https://google.com/remote",
				Name:     "google",
				Revision: "125",
			},
		},
		Projects: []Project{
			{Path: "baz/", Name: "baz", RemoteName: "chromium"},
			{Path: "fiz/", Name: "fiz", Revision: "124"},
			{Name: "buz", RemoteName: "google",
				Annotations: []Annotation{
					{Name: "branch-mode", Value: "pin"},
				},
			},
		},
		Includes: []Include{
			{"bar.xml"},
		},
	}

	expected := &Manifest{
		Projects: []Project{
			{Path: "baz/", Name: "baz", Revision: "123", RemoteName: "chromium"},
			{Path: "fiz/", Name: "fiz", Revision: "124", RemoteName: "chromeos"},
			{Path: "buz", Name: "buz", Revision: "125", RemoteName: "google",
				Annotations: []Annotation{
					{Name: "branch-mode", Value: "pin"},
				},
			},
		},
		Includes: []Include{
			{"bar.xml"},
		},
	}

	manifest.ResolveImplicitLinks()
	assert.Assert(t, ManifestEq(manifest, expected))
}

func TestGetUniqueProject(t *testing.T) {
	manifest := &Manifest{
		Projects: []Project{
			{Path: "foo-a/", Name: "foo"},
			{Path: "foo-b/", Name: "foo"},
			{Path: "bar/", Name: "bar"},
		},
	}

	_, err := manifest.GetUniqueProject("foo")
	assert.ErrorContains(t, err, "multiple projects")

	project, err := manifest.GetUniqueProject("bar")
	assert.NilError(t, err)
	assert.Assert(t, reflect.DeepEqual(&project, &manifest.Projects[2]))
}

func TestWrite(t *testing.T) {
	tmpDir := "repotest_tmp_dir"
	tmpDir, err := ioutil.TempDir("", tmpDir)
	defer os.RemoveAll(tmpDir)
	assert.NilError(t, err)
	tmpPath := filepath.Join(tmpDir, "foo.xml")

	manifest := &Manifest{
		Projects: []Project{
			{Path: "foo-a/", Name: "foo"},
			{Path: "foo-b/", Name: "foo"},
			{Path: "bar/", Name: "bar"},
		},
	}
	manifest.Write(tmpPath)
	// Make sure file was written successfully.
	_, err = os.Stat(tmpPath)
	assert.NilError(t, err)
	// Make sure manifest was marshalled and written correctly.

	data, err := ioutil.ReadFile(tmpPath)
	assert.NilError(t, err)

	got := &Manifest{}
	assert.NilError(t, xml.Unmarshal(data, got))
	assert.Assert(t, ManifestEq(manifest, got))
}

func TestGitName(t *testing.T) {
	remote := Remote{
		Alias: "batman",
		Name:  "bruce wayne",
	}
	assert.StringsEqual(t, remote.GitName(), "batman")
	remote = Remote{
		Name: "robin",
	}
	assert.StringsEqual(t, remote.GitName(), "robin")
}

func TestGetProjectByName(t *testing.T) {
	m := Manifest{
		Projects: []Project{
			{Path: "a/", Name: "a"},
			{Path: "b/", Name: "b"},
			{Path: "c/", Name: "c"},
		},
	}

	project, err := m.GetProjectByName("b")
	assert.NilError(t, err)
	assert.Assert(t, reflect.DeepEqual(*project, m.Projects[1]))
	project, err = m.GetProjectByName("d")
	assert.Assert(t, err != nil)
}
func TestGetProjectByPath(t *testing.T) {
	m := Manifest{
		Projects: []Project{
			{Path: "a/", Name: "a"},
			{Path: "b/", Name: "b"},
		},
	}

	project, err := m.GetProjectByPath("b/")
	assert.NilError(t, err)
	assert.Assert(t, reflect.DeepEqual(*project, m.Projects[1]))

	// Add a project after the fact to test the internal mapping.
	m.Projects = append(m.Projects, Project{Path: "c/", Name: "c"})
	project, err = m.GetProjectByPath("c/")
	assert.NilError(t, err)
	assert.Assert(t, reflect.DeepEqual(*project, m.Projects[2]))

	project, err = m.GetProjectByPath("d/")
	assert.Assert(t, err != nil)
}

func deref(projects []*Project) []Project {
	res := []Project{}
	for _, project := range projects {
		res = append(res, *project)
	}
	return res
}

func TestGetProjects(t *testing.T) {
	m := Manifest{
		Projects: []Project{
			{Path: "a1/", Name: "chromiumos/a"},
			{Path: "a2/", Name: "chromiumos/a", Annotations: []Annotation{{Name: "branch-mode", Value: "pin"}}},
			{Path: "b/", Name: "b", Annotations: []Annotation{{Name: "branch-mode", Value: "pin"}}},
			{Path: "c/", Name: "c", Annotations: []Annotation{{Name: "branch-mode", Value: "tot"}}},
			{Path: "d/", Name: "chromiumos/d"},
			{Path: "e/", Name: "chromiumos/e"},
		},
		Remotes: []Remote{
			{Name: "cros"},
		},
		Default: Default{
			RemoteName: "cros",
		},
	}
	singleProjects := deref(m.GetSingleCheckoutProjects())
	assert.Assert(t, reflect.DeepEqual(singleProjects, m.Projects[4:6]))
	multiProjects := deref(m.GetMultiCheckoutProjects())
	assert.Assert(t, reflect.DeepEqual(multiProjects, m.Projects[:2]))
	pinnedProjects := deref(m.GetPinnedProjects())
	assert.Assert(t, reflect.DeepEqual(pinnedProjects, m.Projects[1:3]))
	totProjects := deref(m.GetTotProjects())
	assert.Assert(t, reflect.DeepEqual(totProjects, m.Projects[3:4]))
}

var canBranchTestManifestAnnotation = Manifest{
	Projects: []Project{
		// Projects with annotations labeling branch mode.
		{Path: "foo1/", Name: "foo1",
			Annotations: []Annotation{
				{Name: "branch-mode", Value: "create"},
			},
		},
		{Path: "foo2/", Name: "foo2",
			Annotations: []Annotation{
				{Name: "branch-mode", Value: "pin"},
			},
		},
		{Path: "foo3/", Name: "foo3",
			Annotations: []Annotation{
				{Name: "branch-mode", Value: "tot"},
			},
		},
		{Path: "foo4/", Name: "foo4",
			Annotations: []Annotation{
				{Name: "branch-mode", Value: "bogus"},
			},
		},
	},
}
var canBranchTestManifestRemote = Manifest{
	Projects: []Project{
		// Remote has name but no alias. Project is branchable.
		{Path: "bar/", Name: "chromiumos/bar", RemoteName: "cros"},
		// Remote has alias. Project is branchable.
		{Path: "baz1/", Name: "aosp/baz", RemoteName: "cros1"},
		// Remote has alias. Remote is not a cros remote.
		{Path: "baz2/", Name: "aosp/baz", RemoteName: "cros2"},
		// Remote has alias. Remote is a cros remote, but not a branchable one.
		{Path: "fizz/", Name: "fizz", RemoteName: "cros"},
		// Remote has name but no alias. Remote is a branchable remote, but specific
		// project is not branchable.
		{Path: "buzz/", Name: "buzz", RemoteName: "weave"},
	},
	Remotes: []Remote{
		{Name: "cros"},
		{Name: "cros1", Alias: "cros"},
		{Name: "cros2", Alias: "github"},
		{Name: "weave"},
	},
}

func assertBranchModesEqual(t *testing.T, a, b BranchMode) {
	assert.StringsEqual(t, string(a), string(b))
}

func TestProjectBranchMode_annotation(t *testing.T) {
	manifest := canBranchTestManifestAnnotation
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[0]), Create)
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[1]), Pinned)
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[2]), Tot)
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[3]), UnspecifiedMode)
}

func TestProjectBranchMode_remote(t *testing.T) {
	manifest := canBranchTestManifestRemote
	// Remote has name but no alias. Project is branchable.
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[0]), Create)
	// Remote has alias. Project is branchable.
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[1]), Create)
	// Remote has alias. Remote is not a cros remote.
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[2]), Pinned)
	// Remote has alias. Remote is a cros remote, but not a branchable one.
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[4]), Pinned)
	// Remote has name but no alias. Remote is a branchable remote, but specific
	// project is not branchable.
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[3]), Pinned)
}

func TestMergeManifests(t *testing.T) {
	// Manifest inheritance is as follows:
	// a --> b --> c
	//  \
	//   \--> d
	a := Manifest{
		Default: Default{
			RemoteName: "cros",
			Revision:   "refs/heads/master",
		},
		Remotes: []Remote{
			{Name: "cros"},
			{Name: "cros-internal"},
		},
		Projects: []Project{
			{Path: "project1/", Name: "project1"},
			{Path: "project2/", Name: "project2"},
			{Path: "project3/", Name: "project3", RemoteName: "cros-internal"},
		},
		Includes: []Include{
			{Name: "b.xml"},
			{Name: "d.xml"},
		},
	}
	b := Manifest{
		Default: Default{
			RemoteName: "cros-internal",
			Revision:   "refs/heads/internal",
		},
		Remotes: []Remote{
			{Name: "cros"},
			{Name: "cros-internal"},
		},
		Projects: []Project{
			{Path: "project3-v2/", Name: "project3"},
			{Path: "project4/", Name: "project4"},
		},
		Includes: []Include{
			{Name: "c.xml"},
		},
	}
	c := Manifest{
		Default: Default{
			RemoteName: "cros-special",
			Revision:   "refs/heads/special",
		},
		Remotes: []Remote{
			{Name: "cros-special", Revision: "refs/heads/unique"},
		},
		Projects: []Project{
			{Path: "project5/", Name: "project5"},
		},
	}
	d := Manifest{
		Default: Default{
			RemoteName: "cros",
			Revision:   "refs/heads/develop",
		},
		Remotes: []Remote{
			{Name: "cros"},
		},
		Projects: []Project{
			{Path: "project6/", Name: "project6"},
			{Path: "project7/", Name: "project7"},
		},
	}
	manifestDict := map[string]*Manifest{
		"a.xml": &a,
		"b.xml": &b,
		"c.xml": &c,
		"d.xml": &d,
	}
	expected := Manifest{
		Default: Default{
			RemoteName: "cros",
			Revision:   "refs/heads/master",
		},
		Remotes: []Remote{
			{Name: "cros"},
			{Name: "cros-internal"},
			{Name: "cros-special", Revision: "refs/heads/unique"},
		},
		Projects: []Project{
			{Path: "project1/", Name: "project1", RemoteName: "cros", Revision: "refs/heads/master"},
			{Path: "project2/", Name: "project2", RemoteName: "cros", Revision: "refs/heads/master"},
			{Path: "project3/", Name: "project3", RemoteName: "cros-internal", Revision: "refs/heads/master"},
			{Path: "project3-v2/", Name: "project3", RemoteName: "cros-internal", Revision: "refs/heads/internal"},
			{Path: "project4/", Name: "project4", RemoteName: "cros-internal", Revision: "refs/heads/internal"},
			{Path: "project5/", Name: "project5", RemoteName: "cros-special", Revision: "refs/heads/unique"},
			{Path: "project6/", Name: "project6", RemoteName: "cros", Revision: "refs/heads/develop"},
			{Path: "project7/", Name: "project7", RemoteName: "cros", Revision: "refs/heads/develop"},
		},
		Includes: []Include{},
	}
	mergedManifest, err := MergeManifests("a.xml", &manifestDict)
	assert.NilError(t, err)
	assert.Assert(t, reflect.DeepEqual(expected, *mergedManifest))
}
