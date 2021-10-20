// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clusteredfailures

import (
	"context"

	bqp "infra/appengine/weetbix/proto/bq"
)

// FakeClient represents a fake implementation of the clustered failures
// exporter, for testing.
type FakeClient struct {
	InsertionsByProject map[string][]*bqp.ClusteredFailureRow
}

// NewFakeClient creates a new FakeClient for exporting clustered failures.
func NewFakeClient() *FakeClient {
	return &FakeClient{
		InsertionsByProject: make(map[string][]*bqp.ClusteredFailureRow),
	}
}

// Insert inserts the given rows in BigQuery.
func (fc *FakeClient) Insert(ctx context.Context, luciProject string, rows []*bqp.ClusteredFailureRow) error {
	inserts := fc.InsertionsByProject[luciProject]
	inserts = append(inserts, rows...)
	fc.InsertionsByProject[luciProject] = inserts
	return nil
}
