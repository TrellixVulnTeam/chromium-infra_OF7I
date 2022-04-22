// Copyright 2018 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fakes

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// GerritClient is a fake implementation of the gitstore.GerritClient interface.
type GerritClient struct {
	nextNumber int64
	Changes    []*GerritChange
}

// GerritChange captures information about a single gerrit change created via
// GerritClient.CreateChange()
type GerritChange struct {
	*gerritpb.ChangeInfo
	GerritChangeEdit
	IsSubmitted bool
}

// GerritChangeEdit captures information about a change edit created via
// GerritClient.ChangeEditFileContent on a ChangeEdit.
type GerritChangeEdit struct {
	// Maps file path to new contents of the file.
	Files       map[string]string
	IsPublished bool
	IsAbandoned bool
	Subject     string
}

// GetChange implements the gerritpb.GerritClient interface.
func (gc *GerritClient) GetChange(ctx context.Context, in *gerritpb.GetChangeRequest, opts ...grpc.CallOption) (*gerritpb.ChangeInfo, error) {
	for _, c := range gc.Changes {
		if in.Number == c.Number {
			return proto.Clone(c.ChangeInfo).(*gerritpb.ChangeInfo), nil
		}
	}
	return nil, fmt.Errorf("No change for %+v", in)
}

// CreateChange implements the gerritpb.GerritClient interface.
func (gc *GerritClient) CreateChange(ctx context.Context, in *gerritpb.CreateChangeRequest, opts ...grpc.CallOption) (*gerritpb.ChangeInfo, error) {
	c := &GerritChange{
		ChangeInfo: &gerritpb.ChangeInfo{
			Number:          gc.nextNumber,
			Project:         in.Project,
			Ref:             in.Ref,
			Status:          gerritpb.ChangeStatus_NEW,
			CurrentRevision: "patch_set_1",
		},
	}
	c.GerritChangeEdit.Files = make(map[string]string)
	c.GerritChangeEdit.Subject = in.Subject
	gc.nextNumber++
	gc.Changes = append(gc.Changes, c)

	return proto.Clone(c.ChangeInfo).(*gerritpb.ChangeInfo), nil
}

// ChangeEditFileContent implements the gerritpb.GerritClient interface.
func (gc *GerritClient) ChangeEditFileContent(ctx context.Context, in *gerritpb.ChangeEditFileContentRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	for _, c := range gc.Changes {
		if in.Number == c.Number {
			c.GerritChangeEdit.Files[in.FilePath] = string(in.Content)
			return &empty.Empty{}, nil
		}
	}
	return &empty.Empty{}, fmt.Errorf("No change edit for %+v", in)
}

// DeleteEditFileContent implements the gerritpb.GerritClient interface.
func (gc *GerritClient) DeleteEditFileContent(ctx context.Context, in *gerritpb.DeleteEditFileContentRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	for _, c := range gc.Changes {
		if in.Number == c.Number {
			c.GerritChangeEdit.Files[in.FilePath] = ""
			return &empty.Empty{}, nil
		}
	}
	return &empty.Empty{}, fmt.Errorf("No change edit for %+v", in)
}

// ChangeEditPublish implements the gerritpb.GerritClient interface.
func (gc *GerritClient) ChangeEditPublish(ctx context.Context, in *gerritpb.ChangeEditPublishRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	for _, c := range gc.Changes {
		if in.Number == c.Number {
			c.GerritChangeEdit.IsPublished = true
			return &empty.Empty{}, nil
		}
	}
	return &empty.Empty{}, fmt.Errorf("No change edit for %+v", in)
}

// SetReview implements the gerritpb.GerritClient interface.
func (gc *GerritClient) SetReview(ctx context.Context, in *gerritpb.SetReviewRequest, opts ...grpc.CallOption) (*gerritpb.ReviewResult, error) {
	// Not needed for tests.
	return &gerritpb.ReviewResult{}, nil
}

// SubmitChange implements the gerritpb.GerritClient interface.
func (gc *GerritClient) SubmitChange(ctx context.Context, in *gerritpb.SubmitChangeRequest, opts ...grpc.CallOption) (*gerritpb.ChangeInfo, error) {
	for _, c := range gc.Changes {
		if in.Number == c.Number {
			c.IsSubmitted = true
			c.ChangeInfo.Status = gerritpb.ChangeStatus_MERGED
			return proto.Clone(c.ChangeInfo).(*gerritpb.ChangeInfo), nil
		}
	}
	return nil, fmt.Errorf("No change for %+v", in)
}

// AbandonChange implements the gerritpb.GerritClient interface.
func (gc *GerritClient) AbandonChange(ctx context.Context, in *gerritpb.AbandonChangeRequest, opts ...grpc.CallOption) (*gerritpb.ChangeInfo, error) {
	for _, c := range gc.Changes {
		if in.Number == c.Number {
			c.IsAbandoned = true
			return proto.Clone(c.ChangeInfo).(*gerritpb.ChangeInfo), nil
		}
	}
	return nil, fmt.Errorf("No change for %+v", in)
}

// GetMergeable implements the gerritpb.GerritClient interface.
func (gc *GerritClient) GetMergeable(ctx context.Context, req *gerritpb.GetMergeableRequest, opts ...grpc.CallOption) (*gerritpb.MergeableInfo, error) {
	return nil, fmt.Errorf("Fake GetMergeable not yet implemented")
}

// ListFiles implements the gerritpb.GerritClient interface.
func (gc *GerritClient) ListFiles(ctx context.Context, req *gerritpb.ListFilesRequest, opts ...grpc.CallOption) (*gerritpb.ListFilesResponse, error) {
	return nil, fmt.Errorf("Fake ListFiles not yet implemented")
}

// ListChanges that match a query.
//
// Note, although the Gerrit API supports multiple queries, for which
// it can return multiple lists of changes, this is not a foreseen use-case
// so this API just includes one query with one returned list of changes.
//
// https://gerrit-review.googlesource.com/Documentation/rest-api-changes.html#list-changes
func (gc *GerritClient) ListChanges(ctx context.Context, in *gerritpb.ListChangesRequest, opts ...grpc.CallOption) (*gerritpb.ListChangesResponse, error) {
	return nil, fmt.Errorf("Fake ListChanges not yet implemented")
}
