// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package repo

import (
	"context"
	"io/ioutil"
	"testing"

	"infra/cros/internal/assert"
	"infra/cros/internal/cmd"

	"github.com/google/go-cmp/cmp"
)

func TestInit(t *testing.T) {
	args := InitArgs{
		ManifestURL:    "foo",
		ManifestBranch: "bar",
		ManifestFile:   "baz",
	}
	CommandRunnerImpl = cmd.FakeCommandRunner{
		ExpectedCmd: []string{"repo", "init",
			"-u", "foo", "-b", "bar", "-m", "baz"},
		ExpectedDir: "pwd",
	}
	assert.NilError(t, Init(context.Background(), "pwd", "repo", args))
}
func TestSync(t *testing.T) {
	CommandRunnerImpl = cmd.FakeCommandRunner{
		ExpectedCmd: []string{"repo", "sync"},
		ExpectedDir: "pwd",
	}
	assert.NilError(t, Sync(context.Background(), "pwd", "repo"))
}

func TestGetRepoToSourceRoot_success(t *testing.T) {
	f, err := ioutil.TempDir("", "repotest_tmp_dir")
	CommandRunnerImpl = cmd.FakeCommandRunner{
		// This is a sample of `repo list` output.
		Stdout: `chromeos-admin : chromeos/chromeos-admin
chromite : chromiumos/chromite
`,
		ExpectedDir: f,
	}
	if err != nil {
		t.Error(err)
	}
	actual, err := GetRepoToSourceRoot(f, "repo")
	if err != nil {
		t.Error(err)
	}
	expected := map[string]string{
		"chromeos/chromeos-admin": "chromeos-admin",
		"chromiumos/chromite":     "chromite",
	}
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("RepoToSourceRoot bad result (-want/+got)\n%s", diff)
	}
}

func TestGetRepoToSourceRoot_repoToolFails(t *testing.T) {
	f, err := ioutil.TempDir("", "repotest_tmp_dir")
	CommandRunnerImpl = cmd.FakeCommandRunner{
		ExpectedDir: f,
		// Simulate the `repo list` command returning a nonzero exit code.
		FailCommand: true,
	}
	if err != nil {
		t.Error(err)
	}
	_, err = GetRepoToSourceRoot(f, "repo")
	if err == nil {
		t.Error("expected an error")
	}
}
