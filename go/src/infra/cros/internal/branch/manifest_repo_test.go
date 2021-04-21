// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package branch

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"infra/cros/internal/assert"
	"infra/cros/internal/cmd"
	"infra/cros/internal/git"
	"infra/cros/internal/repo"
)

func TestRepairManifest_success(t *testing.T) {
	manifestRepo := ManifestRepo{
		Project: repo.Project{
			Name: "chromiumos/manifest",
		},
		ProjectCheckout: "foo",
	}
	branchPathMap := map[string]string{
		"foo/":          "branch",
		"src/repohooks": "branch",
	}
	// Variations in attribute order and singleton tags are to test
	// regexp parsing.
	originalManifest := `
	 <?xml version="1.0" encoding="UTF-8"?>
	 <manifest>
	 	<!---Comment 1-->
	 	<default remote="cros" revision="test" />
	 	<remote name="cros" revision="refs/heads/default"></remote>
	 	<remote revision="124" name="remote2" />
	 	<remote name="remote3" revision="125" />

		<project path="src/repohooks" name="chromiumos/repohooks"
			groups="minilayout,firmware,buildtools,labtools,crosvm" />
		<repo-hooks in-project="chromiumos/repohooks" enabled-list="pre-upload" />

	 	<project name="chromiumos/foo" path="foo/" upstream="" />
	 	<project upstream="" name="pinned" revision="456" path="pinned/">
	 		<!---Comment 2-->
	 		<annotation name="branch-mode" value="pin" />
	 	</project>
	 	<project name="tot" path="tot1/">
	 		<annotation name="branch-mode" value="tot" />
	 	</project>
	 	<project name="tot" path="tot2/" revision="refs/heads/tot">
	 		<annotation name="branch-mode" value="tot" />
	 	</project>
	 </manifest>
	`

	err := xml.Unmarshal([]byte(originalManifest), &WorkingManifest)
	WorkingManifest.ResolveImplicitLinks()

	assert.NilError(t, err)

	// Mock out loadManifestFromFile
	loadManifestFromFileRaw = func(path string) ([]byte, error) {
		return []byte(originalManifest), nil
	}
	git.CommandRunnerImpl = cmd.FakeCommandRunner{
		Stdout: "123 test",
	}

	manifestRaw, err := manifestRepo.repairManifest("fake_path", branchPathMap)
	assert.NilError(t, err)

	manifest := repo.Manifest{}
	assert.NilError(t, xml.Unmarshal(manifestRaw, &manifest))
	// repairManifest deletes revision attr on <default>
	assert.StringsEqual(t, manifest.Default.Revision, "")
	// repairManifest deletes revision attr on <remote>
	for _, remote := range manifest.Remotes {
		assert.StringsEqual(t, remote.Revision, "")
	}
	// repairManifest properly sets revision on branched projects.
	assert.StringsEqual(t, manifest.Projects[0].Revision, "refs/heads/branch")
	assert.StringsEqual(t, manifest.Projects[1].Revision, "refs/heads/branch")
	// repairManifest properly sets revision on pinned projects.
	assert.StringsEqual(t, manifest.Projects[2].Revision, "123")
	// repairManifest properly sets revision on ToT projects.
	assert.StringsEqual(t, manifest.Projects[3].Revision, "refs/heads/default")
	assert.StringsEqual(t, manifest.Projects[4].Revision, "refs/heads/tot")

	// Check that manifest is otherwise unmodified.
	expectedManifest := `
	 <?xml version="1.0" encoding="UTF-8"?>
	 <manifest>
	 	<!---Comment 1-->
	 	<default remote="cros" />
	 	<remote name="cros"></remote>
	 	<remote name="remote2" />
	 	<remote name="remote3" />

		<project path="src/repohooks" name="chromiumos/repohooks"
			groups="minilayout,firmware,buildtools,labtools,crosvm" revision="refs/heads/branch" />
		<repo-hooks in-project="chromiumos/repohooks" enabled-list="pre-upload" />

	 	<project name="chromiumos/foo" path="foo/" revision="refs/heads/branch" />
	 	<project name="pinned" revision="123" path="pinned/">
	 		<!---Comment 2-->
	 		<annotation name="branch-mode" value="pin" />
	 	</project>
	 	<project name="tot" path="tot1/" revision="refs/heads/default">
	 		<annotation name="branch-mode" value="tot" />
	 	</project>
	 	<project name="tot" path="tot2/" revision="refs/heads/tot">
	 		<annotation name="branch-mode" value="tot" />
	 	</project>
	 </manifest>
	`
	assert.StringsEqual(t, string(manifestRaw), expectedManifest)
}

func TestRepairManifestsOnDisk(t *testing.T) {
	// Use actual repo implementations
	loadManifestFromFileRaw = repo.LoadManifestFromFileRaw
	loadManifestTree = repo.LoadManifestTree

	defaultManifest := repo.Manifest{
		Includes: []repo.Include{
			{Name: "full.xml"},
		},
	}
	officialManifest := repo.Manifest{
		Includes: []repo.Include{
			{Name: "full.xml"},
		},
	}
	fullManifest := repo.Manifest{
		Default: repo.Default{
			RemoteName: "cros",
			Revision:   "refs/heads/master",
		},
		Projects: []repo.Project{
			{Name: "chromiumos/foo", Path: "foo/"},
		},
		Remotes: []repo.Remote{
			{Name: "cros", Revision: "123"},
		},
	}
	expectedFullManifest := repo.Manifest{
		Default: repo.Default{
			RemoteName: "cros",
		},
		Projects: []repo.Project{
			{Name: "chromiumos/foo",
				Path:       "foo/",
				Revision:   "refs/heads/newbranch",
				RemoteName: "cros"},
		},
		Remotes: []repo.Remote{
			{Name: "cros", Revision: ""},
		},
	}

	tmpDir := "manifestrepotest_tmp_dir"
	tmpDir, err := ioutil.TempDir("", tmpDir)
	defer os.RemoveAll(tmpDir)
	assert.NilError(t, err)

	manifests := make(map[string]*repo.Manifest)
	manifests["default.xml"] = &defaultManifest
	manifests["official.xml"] = &officialManifest
	manifests["full.xml"] = &fullManifest
	manifestPath := make(map[string]string)

	manifestRepo := ManifestRepo{
		Project: repo.Project{
			Name: tmpDir,
		},
		ProjectCheckout: tmpDir,
	}

	git.CommandRunnerImpl = cmd.FakeCommandRunner{
		Stdout: "123 refs/heads/master",
	}

	// Set up
	for manifestName, manifest := range manifests {
		// Write manifest.
		path := filepath.Join(tmpDir, manifestName)
		manifestPath[manifestName] = path
		assert.NilError(t, manifest.Write(path))
	}

	fooProject := fullManifest.Projects[0]
	branchMap := make(map[string]string)
	branchMap[fooProject.Path] = "newbranch"

	WorkingManifest = fullManifest
	err = manifestRepo.RepairManifestsOnDisk(branchMap)
	assert.NilError(t, err)
	// Read repaired manifests from disk, check expectations.
	defaultManifestMap, err := repo.LoadManifestTree(manifestPath["default.xml"])

	assert.NilError(t, err)
	assert.Assert(t,
		reflect.DeepEqual(expectedFullManifest, *defaultManifestMap["full.xml"]))
	assert.Assert(t,
		reflect.DeepEqual(defaultManifest, *defaultManifestMap["default.xml"]))

	officialManifestMap, err := repo.LoadManifestTree(manifestPath["official.xml"])
	assert.NilError(t, err)
	assert.Assert(t,
		reflect.DeepEqual(officialManifest, *officialManifestMap["official.xml"]))

	git.CommandRunnerImpl = cmd.RealCommandRunner{}
}
