// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bigquery

import "context"

// FakeClient is a fake BigQuery client for use in tests.
type fakeClient struct {
	project string
}

// Project gets the current project of the fake client.
func (c *fakeClient) Project() string {
	return c.project
}

// NewFakeClient produces a new fake client.
func NewFakeClient(ctx context.Context) Client {
	return &fakeClient{
		project: GetProject(ctx),
	}
}
