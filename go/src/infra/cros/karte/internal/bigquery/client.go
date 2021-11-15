// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bigquery

import (
	"context"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"
)

// Client is a BigQuery client, real or fake.
type Client interface {
	Project() string
}

// NewClient produces a new real BigQuery client.
func NewClient(ctx context.Context, opts ...option.ClientOption) (Client, error) {
	return bigquery.NewClient(ctx, GetProject(ctx), opts...)
}
