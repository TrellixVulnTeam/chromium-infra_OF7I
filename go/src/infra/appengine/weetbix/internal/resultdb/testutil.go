// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultdb

import (
	"context"

	"github.com/golang/mock/gomock"

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

// QueryTestVariants mocks the QueryTestVariants RPC.
func (mc *MockedClient) QueryTestVariants(req *rdbpb.QueryTestVariantsRequest, res *rdbpb.QueryTestVariantsResponse) {
	mc.Client.EXPECT().QueryTestVariants(gomock.Any(), proto.MatcherEqual(req),
		gomock.Any()).Return(res, nil)
}

// GetInvocation mocks the GetInvocation RPC.
func (mc *MockedClient) GetInvocation(req *rdbpb.GetInvocationRequest, res *rdbpb.Invocation) {
	mc.Client.EXPECT().GetInvocation(gomock.Any(), proto.MatcherEqual(req),
		gomock.Any()).Return(res, nil)
}

// GetRealm is a shortcut of GetInvocation to get realm of the invocation.
func (mc *MockedClient) GetRealm(inv, realm string) {
	req := &rdbpb.GetInvocationRequest{
		Name: inv,
	}
	mc.GetInvocation(req, &rdbpb.Invocation{
		Name:  inv,
		Realm: realm,
	})
}

// BatchGetTestVariants mocks the BatchGetTestVariants RPC.
func (mc *MockedClient) BatchGetTestVariants(req *rdbpb.BatchGetTestVariantsRequest, res *rdbpb.BatchGetTestVariantsResponse) {
	mc.Client.EXPECT().BatchGetTestVariants(gomock.Any(), proto.MatcherEqual(req),
		gomock.Any()).Return(res, nil)
}
