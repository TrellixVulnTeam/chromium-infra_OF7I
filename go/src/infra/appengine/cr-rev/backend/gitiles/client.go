// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gitiles

import (
	"context"

	gitilesProto "go.chromium.org/luci/common/proto/gitiles"
	"google.golang.org/grpc"
)

type Client interface {
	Log(ctx context.Context, in *gitilesProto.LogRequest, opts ...grpc.CallOption) (out *gitilesProto.LogResponse, err error)
	Refs(ctx context.Context, in *gitilesProto.RefsRequest, opts ...grpc.CallOption) (out *gitilesProto.RefsResponse, err error)
	Archive(ctx context.Context, in *gitilesProto.ArchiveRequest, opts ...grpc.CallOption) (out *gitilesProto.ArchiveResponse, err error)
	DownloadFile(ctx context.Context, in *gitilesProto.DownloadFileRequest, opts ...grpc.CallOption) (out *gitilesProto.DownloadFileResponse, err error)
	Projects(ctx context.Context, in *gitilesProto.ProjectsRequest, opts ...grpc.CallOption) (out *gitilesProto.ProjectsResponse, err error)
}

var clientKey = "gitiles.client"

// GetClient returns gitilesClient from context. If not set, it panics.
func GetClient(ctx context.Context) Client {
	v := ctx.Value(&clientKey)
	if v == nil {
		panic("Client is not set, use SetClient")
	}

	return v.(Client)

}

// SetClient stores gitilesClient in context. Useful for unit testing.
func SetClient(ctx context.Context, client Client) context.Context {
	return context.WithValue(ctx, &clientKey, client)
}
