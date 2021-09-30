// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package gerrit contains functions for interacting with gerrit/gitiles.
package gerrit

import (
	"io/ioutil"
	"testing"

	"infra/cros/internal/assert"
)

type FakeAPIClient struct {
	T *testing.T
	// ExpectedDownloads is indexed by project, then branch, then file.
	ExpectedDownloads map[string]map[string]map[string]string
	ExpectedProjects  []string
}

// DownloadFileFromGitiles downloads a file from Gitiles.
func (g *FakeAPIClient) DownloadFileFromGitiles(project, branch, path string) (string, error) {
	if projectData, ok := g.ExpectedDownloads[project]; !ok {
		g.T.Fatalf("unexpected download for project %s", project)
	} else if branchData, ok := projectData[branch]; !ok {
		g.T.Fatalf("unexpected download for project %s, branch %s", project, branch)
	} else if fileData, ok := branchData[path]; !ok {
		g.T.Fatalf("unexpected download for project %s, branch %s, file %s", project, branch, path)
	} else {
		return fileData, nil
	}
	return "", nil
}

// DownloadFileFromGitilesToPath downloads a file from Gitiles to a specified path.
func (g *FakeAPIClient) DownloadFileFromGitilesToPath(project, branch, path, saveToPath string) error {
	data, _ := g.DownloadFileFromGitiles(project, branch, path)

	assert.NilError(g.T, ioutil.WriteFile(saveToPath, []byte(data), 0644))
	return nil
}

// Projects returns a list of projects.
func (c *FakeAPIClient) Projects() ([]string, error) {
	return c.ExpectedProjects, nil
}
