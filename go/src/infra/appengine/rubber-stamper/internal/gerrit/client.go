// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gerrit

import (
	"context"

	"google.golang.org/grpc"

	"go.chromium.org/luci/common/errors"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
)

// Client defines a subset of Gerrit API used by rubber-stamper.
type Client interface {
	CLReaderClient
}

// ClientFactory creates Client tied to Gerrit host and LUCI project.
type ClientFactory func(ctx context.Context, gerritHost string) (Client, error)

// Client must be a subset of gerritpb.Client
var _ Client = (gerritpb.GerritClient)(nil)

var clientCtxKey = "infra/appengine/rubber-stamper/internal/client/gerrit.Client"

// setClientFactory puts a given ClientFactory into in the context.
func setClientFactory(ctx context.Context, f ClientFactory) context.Context {
	return context.WithValue(ctx, &clientCtxKey, f)
}

// Setup puts a production ClientFactory into the context.
func Setup(ctx context.Context) context.Context {
	return setClientFactory(ctx, newFactory().makeClient)
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
}
