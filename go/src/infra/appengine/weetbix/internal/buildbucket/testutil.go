// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package buildbucket

import (
	"context"

	"github.com/golang/mock/gomock"

	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/proto"
)

// MockedClient is a mocked Buildbucket client for testing.
// It wraps a bbpb.MockBuildsClient and a context with the mocked client.
type MockedClient struct {
	Client *bbpb.MockBuildsClient
	Ctx    context.Context
}

// NewMockedClient creates a MockedClient for testing.
func NewMockedClient(ctx context.Context, ctl *gomock.Controller) *MockedClient {
	mockClient := bbpb.NewMockBuildsClient(ctl)
	return &MockedClient{
		Client: mockClient,
		Ctx:    context.WithValue(ctx, &mockedBBClientKey, mockClient),
	}
}

// GetBuild Mocks the GetBuild RPC.
func (mc *MockedClient) GetBuild(req *bbpb.GetBuildRequest, res *bbpb.Build) {
	mc.Client.EXPECT().GetBuild(gomock.Any(), proto.MatcherEqual(req), gomock.Any()).Return(res, nil)
}
