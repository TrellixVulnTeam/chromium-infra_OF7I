// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package branch

import (
	"testing"

	"infra/cros/internal/assert"
	"infra/cros/internal/cmd"
	"infra/cros/internal/git"
	"infra/cros/internal/repo"
)

func TestProjectFetchURL(t *testing.T) {
	WorkingManifest = repo.Manifest{
		Remotes: []repo.Remote{
			{Name: "remote", Fetch: "file:///tmp/path/to/remote"},
		},
		Projects: []repo.Project{
			{Path: "foo/bar/project", Name: "foo/bar/project", RemoteName: "remote"},
		},
	}
	url, err := ProjectFetchURL("foo/bar/project")
	assert.NilError(t, err)
	assert.StringsEqual(t, url, "file:///tmp/path/to/remote/foo/bar/project")
}

func TestGetProjectCheckoutFromUrl(t *testing.T) {
	git.CommandRunnerImpl = &cmd.FakeCommandRunnerMulti{
		CommandRunners: []cmd.FakeCommandRunner{
			{
				ExpectedDir: "",
				ExpectedCmd: []string{"git", "init"},
			},
			{
				ExpectedDir: "",
				ExpectedCmd: []string{"git", "remote", "add", "origin", "localhost"},
			},
			{
				ExpectedDir: "",
				ExpectedCmd: []string{"git", "fetch", "origin"},
			},
			{
				ExpectedDir: "",
				ExpectedCmd: []string{"git", "ls-remote", "-q", "--symref", "--exit-code", "origin", "HEAD"},
				Stdout:      "ref: refs/heads/mamama\tHEAD\n5f6803b100bb3cd0f534e96e88c91373e8ed1c44\tHEAD\n",
			},
			{
				ExpectedDir: "",
				ExpectedCmd: []string{"git", "checkout", "mamama"},
			},
		},
	}

	_, err := getProjectCheckoutFromURL("localhost", nil)

	if err != nil {
		t.Error("Error: checkout out project reason: ", err.Error())
		return
	}
}
