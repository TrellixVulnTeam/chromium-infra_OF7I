// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package gerrit contains functions for interacting with gerrit/gitiles.
package gerrit

import (
	"bytes"
	"context"
	gerrs "errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"infra/cros/internal/cmd"
	"infra/cros/internal/git"
	"infra/cros/internal/shared"

	gerritapi "github.com/andygrunwald/go-gerrit"
	"go.chromium.org/luci/common/errors"
)

// This file contains support for interacting with the Gerrit REST API client.

type APIClient interface {
	Projects() ([]string, error)
	DownloadFileFromGitiles(project, branch, path string) (string, error)
	DownloadFileFromGitilesToPath(project, branch, path, saveToPath string) error
}

type ProdAPIClient struct {
	innerClient *gerritapi.Client
}

func NewProdAPIClient(ctx context.Context, host, gitcookiesPath string) (*ProdAPIClient, error) {
	client, err := gerritapi.NewClient(host, nil)
	if err != nil {
		return nil, err
	}
	bareHost := strings.TrimPrefix(host, "http://")
	bareHost = strings.TrimPrefix(host, "https://")

	cmdRunner := cmd.RealCommandRunner{}
	cmd := []string{bareHost, gitcookiesPath}
	var stdoutBuf, stderrBuf bytes.Buffer
	if err := cmdRunner.RunCommand(ctx, &stdoutBuf, &stderrBuf, "", "grep", cmd...); err != nil {
		if e, ok := err.(*exec.ExitError); ok && e.ExitCode() == 1 && strings.HasSuffix(host, "googlesource.com") {
			// grep didn't find a credential for the host.
			// Try looking for a generic .googlesource credential (if the
			// host is a googlesource host).
			stdoutBuf = *bytes.NewBuffer(nil)
			stderrBuf = *bytes.NewBuffer(nil)
			cmd = []string{".googlesource.com", gitcookiesPath}
			if err := cmdRunner.RunCommand(ctx, &stdoutBuf, &stderrBuf, "", "grep", cmd...); err != nil {
				if e, ok := err.(*exec.ExitError); ok && e.ExitCode() == 1 {
					return nil, fmt.Errorf("error reading gitcookies from %s: couldn't find credential for .googlesource.com", gitcookiesPath)
				} else {
					return nil, fmt.Errorf("error reading gitcookies from %s: %s", gitcookiesPath, stderrBuf.String())
				}
			}
		} else {
			return nil, fmt.Errorf("error reading gitcookies from %s: %s", gitcookiesPath, stderrBuf.String())
		}
	}
	cookie := strings.Fields(stdoutBuf.String())
	// Tokenizing on whitespace failed, try commas
	if len(cookie) == 1 {
		cookie = strings.Split(strings.TrimSpace(stdoutBuf.String()), ",")
		if len(cookie) == 1 {
			return nil, fmt.Errorf("failed to tokenize gitcookies from %s, expected whitespace-separated or comma-separated fields", gitcookiesPath)
		}
	}
	cookieKey := cookie[5]
	cookieValue := cookie[6]

	client.Authentication.SetCookieAuth(cookieKey, cookieValue)

	return &ProdAPIClient{
		innerClient: client,
	}, nil
}

// DownloadFileFromGitiles downloads a file from Gitiles.
func (g *ProdAPIClient) DownloadFileFromGitiles(project, branch, path string) (string, error) {
	branch = git.NormalizeRef(branch)
	data, resp, err := g.innerClient.Projects.GetBranchContent(project, branch, path)
	if err != nil {
		if resp.StatusCode == 404 {
			return "", shared.ErrObjectNotExist
		}
		return "", errors.Annotate(err, "download").Err()
	}
	return data, nil
}

// DownloadFileFromGitilesToPath downloads a file from Gitiles to a specified path.
func (g *ProdAPIClient) DownloadFileFromGitilesToPath(project, branch, path, saveToPath string) error {
	contents, err := g.DownloadFileFromGitiles(project, branch, path)
	if err != nil {
		return err
	}

	// Use existing file mode if the file already exists.
	fileMode := os.FileMode(int(0644))
	if fileData, err := os.Stat(saveToPath); err != nil && !gerrs.Is(err, os.ErrNotExist) {
		return err
	} else if fileData != nil {
		fileMode = fileData.Mode()
	}

	return os.WriteFile(saveToPath, []byte(contents), fileMode)
}

// Projects returns a list of projects.
func (c *ProdAPIClient) Projects() ([]string, error) {
	pinfo, _, err := c.innerClient.Projects.ListProjects(&gerritapi.ProjectOptions{
		Prefix: "",
	})
	if err != nil {
		return nil, errors.Annotate(err, "list projects").Err()
	}
	projectNames := make([]string, 0, len(*pinfo))
	for name := range *pinfo {
		projectNames = append(projectNames, name)
	}
	return projectNames, nil
}
