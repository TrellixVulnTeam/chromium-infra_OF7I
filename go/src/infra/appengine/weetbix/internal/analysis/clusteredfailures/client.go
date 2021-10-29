// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clusteredfailures

import (
	"context"

	"cloud.google.com/go/bigquery"
	"go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/server/caching"

	"infra/appengine/weetbix/internal/bqutil"
	bqpb "infra/appengine/weetbix/proto/bq"
)

// The maximum number of rows to insert at a time. With each row not exceeding ~2KB,
// this should keep us clear of the 10MB HTTP request size limit and other limits.
// https://cloud.google.com/bigquery/quotas#all_streaming_inserts
const batchSize = 1000

// tableName is the name of the exported BigQuery table.
const tableName = "clustered_failures"

// schemaApplyer ensures BQ schema matches the row proto definitions.
var schemaApplyer = bq.NewSchemaApplyer(caching.RegisterLRUCache(50))

// NewClient creates a new client for exporting clustered failures.
func NewClient(projectID string) *Client {
	return &Client{
		projectID: projectID,
	}
}

// Client provides methods to export clustered failures to BigQuery.
type Client struct {
	// projectID is the name of the GCP project that contains Weetbix datasets.
	projectID string
}

// Insert inserts the given rows in BigQuery.
func (c *Client) Insert(ctx context.Context, luciProject string, rows []*bqpb.ClusteredFailureRow) error {
	client, err := bqutil.Client(ctx, c.projectID)
	if err != nil {
		return errors.Annotate(err, "creating BQ client").Err()
	}
	defer client.Close()

	dataset, err := bqutil.DatasetForProject(luciProject)
	if err != nil {
		return errors.Annotate(err, "getting dataset").Err()
	}

	// Dataset for the project may have to be manually created.
	table := client.Dataset(dataset).Table(tableName)
	if err := schemaApplyer.EnsureTable(ctx, table, tableMetadata); err != nil {
		return errors.Annotate(err, "ensuring clustered failures table in dataset %q", dataset).Err()
	}

	bqRows := make([]*bq.Row, 0, len(rows))
	for _, r := range rows {
		// bq.Row implements ValueSaver for arbitrary protos.
		bqRow := &bq.Row{
			Message:  r,
			InsertID: bigquery.NoDedupeID,
		}
		bqRows = append(bqRows, bqRow)
	}

	inserter := bqutil.NewInserter(table, batchSize)
	if err := inserter.Put(ctx, bqRows); err != nil {
		return errors.Annotate(err, "inserting clustered failures").Err()
	}
	return nil
}
