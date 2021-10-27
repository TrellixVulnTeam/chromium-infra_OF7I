// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cv

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	cvv0 "go.chromium.org/luci/cv/api/v0"
)

// FakeClient provides a fake implementation of cvv0.RunsClient for testing.
type FakeClient struct {
	Runs map[string]*cvv0.Run
}

func UseFakeClient(ctx context.Context, runs map[string]*cvv0.Run) context.Context {
	return context.WithValue(ctx, &fakeCVClientKey, &FakeClient{Runs: runs})
}

// GetRun mocks cvv0.RunsClient.GetRun RPC.
func (fc *FakeClient) GetRun(ctx context.Context, req *cvv0.GetRunRequest, opts ...grpc.CallOption) (*cvv0.Run, error) {
	if r, ok := fc.Runs[req.Id]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("not found")
}
