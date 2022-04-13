// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gitiles

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"infra/chromium/bootstrapper/gitiles"
	"infra/chromium/util"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/proto/git"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/common/system/filesystem"
	"go.chromium.org/luci/common/testing/testfs"
	"google.golang.org/grpc"
)

type Revision struct {
	// Parent is the commit ID of the parent revision.
	Parent string

	// Files maps file paths to their new contents at the revision.
	//
	// Missing keys will have the same contents as in the parent revision,
	// if there is one, otherwise the file does not exist at the revision. A
	// nil value indicates the file does not exist at the revision.
	Files map[string]*string
}

// Project is the fake data for a gitiles project.
type Project struct {
	// Refs maps refs to their revision.
	//
	// Missing keys will have a default revision computed. An empty string
	// value indicates that the ref does not exist.
	Refs map[string]string

	// Revisions maps commit IDs to the revision of the repo.
	//
	// Missing keys will have a default fake revision. A nil value indicates
	// no revision with the commit ID exists.
	Revisions map[string]*Revision
}

// Host is the fake data for a gitiles host.
type Host struct {
	// Projects maps project names to their details.
	//
	// Missing keys will have a default fake project. A nil value indicates
	// the the project does not exist.
	Projects map[string]*Project
}

// Client is the client that will serve fake data for a given host.
type Client struct {
	hostname string
	gitiles  *Host
}

// Factory creates a factory that returns RPC clients that use fake data to
// respond to requests.
//
// The fake data is taken from the fakes argument, which is a map from host
// names to the Host instances containing the fake data for the host. Missing
// keys will have a default Host. A nil value indicates that the given host is
// not a gitiles instance.
func Factory(fakes map[string]*Host) gitiles.GitilesClientFactory {
	return func(ctx context.Context, host string) (gitiles.GitilesClient, error) {
		fake, ok := fakes[host]
		if !ok {
			fake = &Host{}
		} else if fake == nil {
			return nil, errors.Reason("%s is not a gitiles host", host).Err()
		}
		return &Client{host, fake}, nil
	}
}

func (c *Client) getProject(projectName string) (*Project, error) {
	project, ok := c.gitiles.Projects[projectName]
	if !ok {
		return &Project{}, nil
	} else if project == nil {
		return nil, errors.Reason("unknown project %#v on host %#v", projectName, c.hostname).Err()
	}
	return project, nil
}

func (c *Client) Log(ctx context.Context, request *gitilespb.LogRequest, options ...grpc.CallOption) (*gitilespb.LogResponse, error) {
	util.PanicIf(request.PageSize != 1, "unexpected page_size in LogRequest: %d", request.PageSize)
	project, err := c.getProject(request.Project)
	if err != nil {
		return nil, err
	}
	commitId, ok := project.Refs[request.Committish]
	if !ok {
		commitId = fmt.Sprintf("fake-revision|%s|%s|%s", c.hostname, request.Project, request.Committish)
	} else if commitId == "" {
		return nil, errors.Reason("unknown ref %#v for project %#v on host %#v", request.Committish, request.Project, c.hostname).Err()
	}
	return &gitilespb.LogResponse{
		Log: []*git.Commit{
			{
				Id: commitId,
			},
		},
	}, nil
}

type commit struct {
	id       string
	revision *Revision
}

func (c *Client) getRevisionHistory(projectName, commitId string) ([]*commit, error) {
	project, err := c.getProject(projectName)
	if err != nil {
		return nil, err
	}
	var history []*commit
	for commitId != "" {
		revision, ok := project.Revisions[commitId]
		if !ok {
			revision = &Revision{}
		} else if revision == nil {
			return nil, errors.Reason("unknown revision %#v of project %#v on host %#v", commitId, projectName, c.hostname).Err()
		}
		history = append(history, &commit{commitId, revision})
		commitId = revision.Parent
	}
	return history, nil
}

func (c *Client) DownloadFile(ctx context.Context, request *gitilespb.DownloadFileRequest, options ...grpc.CallOption) (*gitilespb.DownloadFileResponse, error) {
	history, err := c.getRevisionHistory(request.Project, request.Committish)
	if err != nil {
		return nil, err
	}
	var contents *string
	var ok bool
	for _, commit := range history {
		contents, ok = commit.revision.Files[request.Path]
		if ok {
			break
		}
	}
	if contents == nil {
		return nil, errors.Reason("unknown file %#v at revision %#v of project %#v on host %#v", request.Path, request.Committish, request.Project, c.hostname).Err()
	}
	return &gitilespb.DownloadFileResponse{
		Contents: *contents,
	}, nil
}

// DownloadDiff downloads the diff between a revision and its parent.
//
// To ensure that the diffs created are accurate and match the behavior of git
// (which implements its own diffing with rename/copy detection), a local git
// instance is created and commits populated with the fake data. This git
// instance then produces the diff.
func (c *Client) DownloadDiff(ctx context.Context, request *gitilespb.DownloadDiffRequest, options ...grpc.CallOption) (*gitilespb.DownloadDiffResponse, error) {
	history, err := c.getRevisionHistory(request.Project, request.Committish)
	if err != nil {
		return nil, err
	}

	tmp, err := ioutil.TempDir("", "")
	util.PanicOnError(err)

	git := func(args ...string) string {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = tmp
		output, err := cmd.Output()
		util.PanicOnError(err)
		return string(output)
	}
	git("init")

	commit := func(message string) {
		git("add", ".")
		git("commit", "--allow-empty", "-m", message)
	}

	files := map[string]string{}
	for i := len(history) - 1; i > 0; i -= 1 {
		for path, contents := range history[i].revision.Files {
			if contents == nil {
				delete(files, path)
			} else {
				files[path] = *contents
			}
		}
	}
	util.PanicOnError(testfs.Build(tmp, files))
	commit("parent commit")

	for path, contents := range history[0].revision.Files {
		f := filepath.Join(tmp, filepath.FromSlash(path))
		if contents != nil {
			util.PanicOnError(filesystem.MakeDirs(filepath.Dir(f)))
			util.PanicOnError(ioutil.WriteFile(f, []byte(*contents), 0644))
		} else {
			if _, ok := files[path]; ok {
				util.PanicOnError(os.Remove(f))
			}
		}
	}
	commit("target commit")

	args := []string{"diff", "HEAD^", "HEAD"}
	if request.Path != "" {
		args = append(args, "--", request.Path)
	}
	diff := git(args...)
	return &gitilespb.DownloadDiffResponse{Contents: string(diff)}, nil
}
