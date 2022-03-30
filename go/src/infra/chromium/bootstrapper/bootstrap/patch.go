// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"go.chromium.org/luci/common/logging"
)

func patchFile(ctx context.Context, filePath, contents, diff string) (string, error) {
	d, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	parent := path.Dir(filePath)
	if err := os.MkdirAll(path.Join(d, parent), 0755); err != nil {
		return "", err
	}
	f := path.Join(d, filePath)
	if err := ioutil.WriteFile(f, []byte(contents), 0644); err != nil {
		return "", err
	}

	// Needs to indicate patch failure if exit code == 1
	// --unsafe-paths: allows applying patch to files not in a git repo
	// -p1: strips the a/ and b/ from the file paths in the diff
	// --include <path>: Limit the patch to only the specified file
	cmd := exec.CommandContext(ctx, "git", "apply", "--unsafe-paths", "-p1", "--include", filePath)
	cmd.Dir = d
	cmd.Stdin = strings.NewReader(diff)
	if output, err := cmd.CombinedOutput(); err != nil {
		logging.Warningf(ctx, "patch failed with output:\n%s", output)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "", PatchRejected.Apply(err)
		}
		return "", err
	}

	newContents, err := ioutil.ReadFile(f)
	if err != nil {
		return "", err
	}
	return string(newContents), nil
}
