// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gerrit

import (
	"context"
	"fmt"

	"infra/chromium/bootstrapper/gerrit"
	"infra/chromium/util"

	"go.chromium.org/luci/common/errors"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	"google.golang.org/grpc"
)

// Patchset is the fake data for a patchset of a change.
type Patchset struct {
	// Revision is the git revision of the change in the corresponding
	// gitiles host.
	Revision string
}

// Change is the fake data for a gerrit change.
type Change struct {
	// Ref is the target ref for the change.
	Ref string
	// Patchsets maps patchset number (>0) to details of the patchset.
	//
	// Revision info will be generated for each patchset with number up to
	// the greatest key. If no patchsets are provided, the change will have
	// one patchset generated. Any generated patchsets will have a generated
	// revision but affect no files. A nil value will be treated as a
	// default Patchset.
	Patchsets map[int32]*Patchset
}

// Project is the fake data for a gerrit project.
type Project struct {
	Changes map[int64]*Change
}

// Host is the fake data for a gerrit host.
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
	gerrit   *Host
}

// Factory creates a factory that returns RPC clients that use fake data to
// respond to requests.
//
// The fake data is taken from the fakes argument, which is a map from host
// names to the Host instances containing the fake data for the host. Missing
// keys will have a default Host. A nil value indicates that the given host is
// not a gerrit instance.
func Factory(fakes map[string]*Host) gerrit.GerritClientFactory {
	return func(ctx context.Context, host string) (gerrit.GerritClient, error) {
		fake, ok := fakes[host]
		if !ok {
			fake = &Host{}
		} else if fake == nil {
			return nil, errors.Reason("%s is not a gerrit host", host).Err()
		}
		return &Client{host, fake}, nil
	}
}

func (c *Client) getChange(projectName string, changeNumber int64) (*Change, error) {
	project, ok := c.gerrit.Projects[projectName]
	if !ok {
		project = &Project{}
	} else if project == nil {
		return nil, errors.Reason("unknown project %#v on host %#v", projectName, c.hostname).Err()
	}
	change, ok := project.Changes[changeNumber]
	if !ok {
		change = &Change{Ref: fmt.Sprintf("fake-ref|%s|%s|%d", c.hostname, projectName, changeNumber)}
	} else if change == nil {
		return nil, errors.Reason("change %d does not exist for project %#v on host %#v", changeNumber, projectName, c.hostname).Err()
	}
	return change, nil
}

func maxPatchsetNum(c *Change) (max int32) {
	max = 1
	for patchsetNum := range c.Patchsets {
		if patchsetNum > max {
			max = patchsetNum
		}
	}
	return
}

func (c *Client) GetChange(ctx context.Context, request *gerritpb.GetChangeRequest, opts ...grpc.CallOption) (*gerritpb.ChangeInfo, error) {
	util.PanicIf(len(request.Options) != 1 || request.Options[0] != gerritpb.QueryOption_ALL_REVISIONS,
		"unexpected options in GetChange: %v", request.Options)

	change, err := c.getChange(request.Project, request.Number)
	if err != nil {
		return nil, err
	}
	revisions := map[string]*gerritpb.RevisionInfo{}
	max := maxPatchsetNum(change)
	for i := int32(1); i <= max; i += 1 {
		patchset := change.Patchsets[i]
		if patchset == nil {
			patchset = &Patchset{}
		}
		revision := patchset.Revision
		if revision == "" {
			revision = fmt.Sprintf("fake-revision|%s|%s|%d|%d", c.hostname, request.Project, request.Number, i)
		}
		revisions[revision] = &gerritpb.RevisionInfo{Number: i}
	}
	return &gerritpb.ChangeInfo{
		Project:   request.Project,
		Number:    request.Number,
		Ref:       change.Ref,
		Revisions: revisions,
	}, nil
}
