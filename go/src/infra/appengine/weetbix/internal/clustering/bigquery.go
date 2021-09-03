// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clustering

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

// ImpactfulClusterReadOptions specifies options for ReadImpactfulClusters().
type ImpactfulClusterReadOptions struct {
	// The unexpected failure (1 day) threshold, which if met or exceeded,
	// should result in the cluster being returned.
	UnexpectedFailures1dThreshold int
	// The unexpected failure (3 day) threshold, which if met or exceeded,
	// should result in the cluster being read.
	UnexpectedFailures3dThreshold int
	// The unexpected failure (3 day) threshold, which if met or exceeded,
	// should result in the cluster being read.
	UnexpectedFailures7dThreshold int
	// The set of clusters for which analysis should always be read.
	// This is typically the set of clusters for which bugs have been
	// filed.
	AlwaysIncludeClusterIDs []string
}

// Cluster represents a group of failures, with associated impact metrics.
type Cluster struct {
	ClusterID              string
	UnexpectedFailures1d   int
	UnexpectedFailures3d   int
	UnexpectedFailures7d   int
	UnexoneratedFailures1d int
	UnexoneratedFailures3d int
	UnexoneratedFailures7d int
	AffectedRuns1d         int
	AffectedRuns3d         int
	AffectedRuns7d         int
	ExampleFailureReason   bigquery.NullString
}

// NewClient creates a new client for reading clusters. Close() MUST
// be called after you have finished using this client.
func NewClient(ctx context.Context, projectID string) (*Client, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &Client{client: client}, nil
}

// Client may be used to read Weetbix clusters.
type Client struct {
	client *bigquery.Client
}

// Close releases any resources held by the client.
func (c *Client) Close() error {
	return c.client.Close()
}

// ReadImpactfulClusters reads clusters exceeding specified impact metrics, or are otherwise
// nominated to be read.
func (c *Client) ReadImpactfulClusters(ctx context.Context, opts ImpactfulClusterReadOptions) ([]*Cluster, error) {
	q := c.client.Query(`
	SELECT cluster_id as ClusterID,
		unexpected_failures_1d as UnexpectedFailures1d,
		unexpected_failures_3d as UnexpectedFailures3d,
		unexpected_failures_7d as UnexpectedFailures7d,
		unexonerated_failures_1d as UnexoneratedFailures1d,
		unexonerated_failures_3d as UnexoneratedFailures3d,
		unexonerated_failures_7d as UnexoneratedFailures7d,
		affected_runs_1d as AffectedRuns1d,
		affected_runs_3d as AffectedRuns3d,
		affected_runs_7d as AffectedRuns7d,
		example_failure_reason.primary_error_message as ExampleFailureReason
	FROM chromium.clusters
	WHERE (unexpected_failures_1d > @unexpFailThreshold1d
		OR unexpected_failures_3d > @unexpFailThreshold3d
		OR unexpected_failures_7d > @unexpFailThreshold7d)
		OR cluster_id IN UNNEST(@alwaysSelectClusters)
	`)
	// TODO(crbug.com/1243174): This will not scale if the set of
	// cluster IDs to always select grows too large.
	q.Parameters = []bigquery.QueryParameter{
		{Name: "unexpFailThreshold1d", Value: opts.UnexpectedFailures1dThreshold},
		{Name: "unexpFailThreshold3d", Value: opts.UnexpectedFailures3dThreshold},
		{Name: "unexpFailThreshold7d", Value: opts.UnexpectedFailures7dThreshold},
		{Name: "alwaysSelectClusters", Value: opts.AlwaysIncludeClusterIDs},
	}
	job, err := q.Run(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "querying clusters").Err()
	}
	it, err := job.Read(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "obtain cluster iterator").Err()
	}
	var clusters []*Cluster
	for {
		row := &Cluster{}
		err := it.Next(row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, errors.Annotate(err, "obtain next cluster row").Err()
		}
		clusters = append(clusters, row)
	}
	return clusters, nil
}
