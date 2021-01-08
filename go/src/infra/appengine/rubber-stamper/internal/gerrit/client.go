// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gerrit

import (
	"context"
	"net/http"

	emptypb "github.com/golang/protobuf/ptypes/empty"
	"go.chromium.org/luci/common/api/gerrit"
	"go.chromium.org/luci/common/errors"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	"go.chromium.org/luci/server/auth"
	"google.golang.org/grpc"
)

// Client defines a subset of Gerrit API used by rubber-stamper.
type Client interface {
	CLReaderClient
	CLWriterClient
}

// ClientFactory creates Client tied to Gerrit host and LUCI project.
type ClientFactory func(ctx context.Context, gerritHost string) (Client, error)

// Client must be a subset of gerritpb.Client
var _ Client = (gerritpb.GerritClient)(nil)

var clientCtxKey = "infra/appengine/rubber-stamper/internal/client/gerrit.Client"
var gerritScope = "https://www.googleapis.com/auth/gerritcodereview"

// setClientFactory puts a given ClientFactory into in the context.
func setClientFactory(ctx context.Context, f ClientFactory) context.Context {
	return context.WithValue(ctx, &clientCtxKey, f)
}

// Setup puts a production ClientFactory into the context.
func Setup(ctx context.Context) context.Context {
	return setClientFactory(ctx, func(ctx context.Context, gerritHost string) (Client, error) {
		t, err := auth.GetRPCTransport(ctx, auth.AsSelf, auth.WithScopes(gerritScope))
		if err != nil {
			return nil, err
		}
		return gerrit.NewRESTClient(&http.Client{Transport: t}, gerritHost, true)
	})
}

// SetTestClientFactory sets up a ClientFactory for testing, where clientMap is
// a map whose keys are gerrit hosts, values are corresponding testing Gerrit
// clients.
func SetTestClientFactory(ctx context.Context, clientMap map[string]Client) context.Context {
	return setClientFactory(ctx, func(ctx context.Context, gerritHost string) (Client, error) {
		client, ok := clientMap[gerritHost]
		if !ok {
			return nil, errors.New("not a valid Gerrit host name")
		}
		return client, nil
	})
}

// GetCurrentClient returns the Client in the context or an error.
func GetCurrentClient(ctx context.Context, gerritHost string) (Client, error) {
	f, _ := ctx.Value(&clientCtxKey).(ClientFactory)
	if f == nil {
		return nil, errors.New("not a valid Gerrit context, no ClientFactory available")
	}
	return f(ctx, gerritHost)
}

// CLReaderClient defines a subset of Gerrit API used by rubber-stamper to
// fetch CL details.
type CLReaderClient interface {
	// Lists changes that match a query.
	//
	// https://gerrit-review.googlesource.com/Documentation/rest-api-changes.html#list-changes
	ListChanges(ctx context.Context, in *gerritpb.ListChangesRequest, opts ...grpc.CallOption) (*gerritpb.ListChangesResponse, error)
	// Lists the files that were modified, added or deleted in a revision.
	//
	// https://gerrit-review.googlesource.com/Documentation/rest-api-changes.html#list-files
	ListFiles(ctx context.Context, in *gerritpb.ListFilesRequest, opts ...grpc.CallOption) (*gerritpb.ListFilesResponse, error)
	// Check if the given change is a pure revert of the change it references in
	// revertOf. See also ChangeInfo.revert_of.
	//
	// https://gerrit-review.googlesource.com/Documentation/rest-api-changes.html#get-pure-revert
	GetPureRevert(ctx context.Context, in *gerritpb.GetPureRevertRequest, opts ...grpc.CallOption) (*gerritpb.PureRevertInfo, error)
	// Loads a change by id.
	//
	// https://gerrit-review.googlesource.com/Documentation/rest-api-changes.html#get-change
	GetChange(ctx context.Context, in *gerritpb.GetChangeRequest, opts ...grpc.CallOption) (*gerritpb.ChangeInfo, error)
}

// CLWriterClient defines a subset of Gerrit API used by rubber-stamper to
// review CLs.
type CLWriterClient interface {
	// Set various review bits on a change.
	//
	// https://gerrit-review.googlesource.com/Documentation/rest-api-changes.html#set-review
	SetReview(ctx context.Context, in *gerritpb.SetReviewRequest, opts ...grpc.CallOption) (*gerritpb.ReviewResult, error)
	// Deletes a reviewer from a change.
	//
	// https://gerrit-review.googlesource.com/Documentation/rest-api-changes.html#delete-reviewer
	DeleteReviewer(ctx context.Context, in *gerritpb.DeleteReviewerRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
}
