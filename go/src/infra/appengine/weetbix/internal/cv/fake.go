// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cv

import (
	"context"

	cvv0 "go.chromium.org/luci/cv/api/v0"
	"google.golang.org/grpc"
)

// FakeClient provides a fake implementation of cvv0.RunsClient for testing.
type FakeClient struct {
}

func UseFakeClient(ctx context.Context) context.Context {
	return context.WithValue(ctx, &fakeCVClientKey, &FakeClient{})
}

// GetRun mocks cvv0.RunsClient.GetRun RPC.
func (fc *FakeClient) GetRun(ctx context.Context, req *cvv0.GetRunRequest, opts ...grpc.CallOption) (*cvv0.Run, error) {
	return &cvv0.Run{}, nil
}
