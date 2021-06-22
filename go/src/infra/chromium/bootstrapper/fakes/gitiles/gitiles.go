// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gitiles

import (
	"context"
	"fmt"
	"infra/chromium/bootstrapper/gitiles"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/proto/git"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
	"google.golang.org/grpc"
)

// FileRevId is an identifier for a file at a given git revision.
type FileRevId struct {
	// Revision is the git revision.
	Revision string
	// Path is the path to the file.
	Path string
}

// Project is the fake data for a gitiles project.
type Project struct {
	// Refs maps refs to their revision.
	//
	// Missing keys will have a default revision computed. An empty string value
	// indicates that the ref does not exist.
	Refs map[string]string

	// Files maps file revision IDs to the contents.
	//
	// Missing keys indicate the file does not exist at the revision.
	Files map[FileRevId]*string
}

// Host is the fake data for a gitiles host.
type Host struct {
	// Projects maps project names to their details.
	//
	// Missing keys will have a default fake project. A nil value indicates the
	// the project does not exist.
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
	if ok && project == nil {
		return nil, errors.Reason("unknown project %#v on host %#v", projectName, c.hostname).Err()
	}
	return project, nil
}

func (c *Client) Log(ctx context.Context, request *gitilespb.LogRequest, options ...grpc.CallOption) (*gitilespb.LogResponse, error) {
	if request.PageSize != 1 {
		panic(errors.Reason("unexpected page_size in LogRequest: %d", request.PageSize))
	}
	var revision string
	if project, err := c.getProject(request.Project); err != nil {
		return nil, err
	} else if project != nil {
		var ok bool
		revision, ok = project.Refs[request.Committish]
		if ok && revision == "" {
			return nil, errors.Reason("unknown ref %#v for project %#v on host %#v", request.Committish, request.Project, c.hostname).Err()
		}
	}
	if revision == "" {
		revision = fmt.Sprintf("fake-revision|%s|%s|%s", c.hostname, request.Project, request.Committish)
	}
	return &gitilespb.LogResponse{
		Log: []*git.Commit{
			{
				Id: revision,
			},
		},
	}, nil
}

func (c *Client) DownloadFile(ctx context.Context, request *gitilespb.DownloadFileRequest, options ...grpc.CallOption) (*gitilespb.DownloadFileResponse, error) {
	if request.Format != gitilespb.DownloadFileRequest_TEXT {
		panic(errors.Reason("unexpected format in DownloadRequest: %v", request.Format))
	}
	var contents *string
	if project, err := c.getProject(request.Project); err != nil {
		return nil, err
	} else if project != nil {
		var ok bool
		contents, ok = project.Files[FileRevId{request.Committish, request.Path}]
		if ok && contents == nil {
			return nil, errors.Reason("unknown file %#v at revision %#v of project %#v on host %#v", request.Path, request.Committish, request.Project, c.hostname).Err()
		}
	}
	if contents == nil {
		s := fmt.Sprintf(`{
			"$fake-contents": {
				"host": %#v,
				"project": %#v,
				"revision": %#v,
				"path": %#v
			}
		}`, c.hostname, request.Project, request.Committish, request.Path)
		contents = &s
	}
	return &gitilespb.DownloadFileResponse{
		Contents: *contents,
	}, nil
}
