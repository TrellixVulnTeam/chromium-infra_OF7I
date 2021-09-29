// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultdb

import (
	"context"

	"github.com/golang/mock/gomock"
	"google.golang.org/grpc"

	"go.chromium.org/luci/common/proto"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
)

// MockedClient is a mocked ResultDB client for testing.
// It wraps a rdbpb.MockResultDBClient and a context with the mocked client.
type MockedClient struct {
	Client *rdbpb.MockResultDBClient
	Ctx    context.Context
}

// NewMockedClient creates a MockedClient for testing.
func NewMockedClient(ctx context.Context, ctl *gomock.Controller) *MockedClient {
	mockClient := rdbpb.NewMockResultDBClient(ctl)
	return &MockedClient{
		Client: mockClient,
		Ctx:    context.WithValue(ctx, &mockResultDBClientKey, mockClient),
	}
}

// QueryTestVariants Mocks the QueryTestVariants RPC.
func (mc *MockedClient) QueryTestVariants(req *rdbpb.QueryTestVariantsRequest, resF func(ctx context.Context, in *rdbpb.QueryTestVariantsRequest, opt grpc.CallOption) (*rdbpb.QueryTestVariantsResponse, error)) {
	mc.Client.EXPECT().QueryTestVariants(gomock.Any(), proto.MatcherEqual(req),
		gomock.Any()).DoAndReturn(resF)
}
