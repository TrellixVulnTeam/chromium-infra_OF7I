// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bqutil

import (
	"context"
	"net/http"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"

	"go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/server/auth"
)

// Client returns a new BigQuery client for use with the given GCP project,
// that authenticates as the given LUCI Project.
func Client(ctx context.Context, luciProject string, gcpProject string) (*bigquery.Client, error) {
	tr, err := auth.GetRPCTransport(ctx, auth.AsProject, auth.WithProject(luciProject), auth.WithScopes(bigquery.Scope))
	if err != nil {
		return nil, err
	}
	return bigquery.NewClient(ctx, gcpProject, option.WithHTTPClient(&http.Client{
		Transport: tr,
	}))
}

// Inserter provides methods to insert rows into a BigQuery table.
type Inserter struct {
	table     *bigquery.Table
	batchSize int
}

// NewInserter initialises a new inserter.
func NewInserter(table *bigquery.Table, batchSize int) *Inserter {
	return &Inserter{
		table:     table,
		batchSize: batchSize,
	}
}

// Put inserts the given rows into BigQuery.
func (i *Inserter) Put(ctx context.Context, rows []*bq.Row) error {
	inserter := i.table.Inserter()
	for i, batch := range i.batch(rows) {
		if err := inserter.Put(ctx, batch); err != nil {
			return errors.Annotate(err, "putting batch %v", i).Err()
		}
	}
	return nil
}

// batch divides the rows to be inserted into batches of at most batchSize.
func (i *Inserter) batch(rows []*bq.Row) [][]*bq.Row {
	var result [][]*bq.Row
	pages := (len(rows) + (i.batchSize - 1)) / i.batchSize
	for p := 0; p < pages; p++ {
		start := p * i.batchSize
		end := start + i.batchSize
		if end > len(rows) {
			end = len(rows)
		}
		page := rows[start:end]
		result = append(result, page)
	}
	return result
}
