// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package repo

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type fakeCommandRunner struct {
	stdout             string
	stderr             string
	expectedWorkingDir string
	failCommand        bool
}

func (c fakeCommandRunner) runCommand(ctx context.Context, stdoutBuf, stderrBuf *bytes.Buffer, name string, args ...string) error {
	stdoutBuf.WriteString(c.stdout)
	stderrBuf.WriteString(c.stderr)
	if c.expectedWorkingDir != "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		if wd != c.expectedWorkingDir {
			return fmt.Errorf("wrong working directory; expected %s got %s", c.expectedWorkingDir, wd)
		}
	}
	if c.failCommand {
		return &exec.ExitError{}
	}
	return nil
}

func TestGetRepoToSourceRoot_success(t *testing.T) {
	f, err := ioutil.TempDir("", "repotest_tmp_dir")
	commandRunnerImpl = fakeCommandRunner{
		// This is a sample of `repo list` output.
		stdout: `chromeos-admin : chromeos/chromeos-admin
chromite : chromiumos/chromite
`,
		expectedWorkingDir: f,
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
	commandRunnerImpl = fakeCommandRunner{
		expectedWorkingDir: f,
		// Simulate the `repo list` command returning a nonzero exit code.
		failCommand: true,
	}
	if err != nil {
		t.Error(err)
	}
	_, err = GetRepoToSourceRoot(f, "repo")
	if err == nil {
		t.Error("expected an error")
	}
	_, ok := err.(*exec.ExitError)
	if !ok {
		t.Errorf("expected err to be an instance of ExitError, instead got %v", err.Error())
	}
}
