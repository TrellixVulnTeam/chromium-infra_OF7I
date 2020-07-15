// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package fake

import (
	"context"
	"io/ioutil"

	"go.chromium.org/luci/common/errors"
)

// GitClient mocks the git.ClientInterface
type GitClient struct {
}

// GetFile mocks git.ClientInterface.GetFile()
func (gc *GitClient) GetFile(ctx context.Context, path string) (string, error) {
	if path == "test_git_path" {
		return GitData("../frontend/fake/dhcp_test.conf")
	}
	return "", errors.Reason("Unspecified mock path %s", path).Err()
}

// SwitchProject mocks git.ClientInterface.SwitchProject()
func (gc *GitClient) SwitchProject(ctx context.Context, project string) error {
	return nil
}

// GitData mocks a git file content based on a given filepath
func GitData(path string) (string, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
