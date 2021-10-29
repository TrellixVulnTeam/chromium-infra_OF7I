// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package analysis

import (
	"context"
	"fmt"
	"math"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	"go.chromium.org/luci/common/errors"

	"infra/appengine/weetbix/internal/bqutil"
	"infra/appengine/weetbix/internal/config"
)

// ImpactfulClusterReadOptions specifies options for ReadImpactfulClusters().
type ImpactfulClusterReadOptions struct {
	// Project is the LUCI Project for which analysis is being performed.
	Project string
	// Thresholds is the set of thresholds, which if any are met
	// or exceeded, should result in the cluster being returned.
	Thresholds *config.ImpactThreshold
}

// ClusterSummary represents a statistical summary of a cluster's failures,
// and their impact.
type ClusterSummary struct {
	ClusterAlgorithm     string              `json:"clusterAlgorithm"`
	ClusterID            string              `json:"clusterId"`
	PresubmitRejects1d   Counts              `json:"presubmitRejects1d"`
	PresubmitRejects3d   Counts              `json:"presubmitRejects3d"`
	PresubmitRejects7d   Counts              `json:"presubmitRejects7d"`
	TestRunFails1d       Counts              `json:"testRunFailures1d"`
	TestRunFails3d       Counts              `json:"testRunFailures3d"`
	TestRunFails7d       Counts              `json:"testRunFailures7d"`
	Failures1d           Counts              `json:"failures1d"`
	Failures3d           Counts              `json:"failures3d"`
	Failures7d           Counts              `json:"failures7d"`
	AffectedTests1d      []SubCluster        `json:"affectedTests1d"`
	AffectedTests3d      []SubCluster        `json:"affectedTests3d"`
	AffectedTests7d      []SubCluster        `json:"affectedTests7d"`
	ExampleFailureReason bigquery.NullString `json:"exampleFailureReason"`
	ExampleTestID        string              `json:"exampleTestId"`
}

// SubCluster represents the name of a test and the number of times
// a failure has impacted it.
type SubCluster struct {
	Value     string `json:"value"`
	Num_Fails int    `json:"numFails"`
}

// Counts captures the values of an integer-valued metric in different
// calculation bases.
type Counts struct {
	// The statistic value after impact has been reduced by exoneration.
	Nominal int64 `json:"nominal"`
	// The statistic value before impact has been reduced by exoneration.
	PreExononeration int64 `json:"preExoneration"`
	// The statistic value:
	// - excluding impact already counted under other higher-priority clusters
	//   (I.E. bug clusters.)
	// - after impact has been reduced by exoneration.
	Residual int64 `json:"residual"`
	// The statistic value:
	// - excluding impact already counted under other higher-priority clusters
	//   (I.E. bug clusters.)
	// - before impact has been reduced by exoneration.
	ResidualPreExoneration int64 `json:"residualPreExoneration"`
}

// NewClient creates a new client for reading clusters. Close() MUST
// be called after you have finished using this client.
func NewClient(ctx context.Context, gcpProject string) (*Client, error) {
	client, err := bqutil.Client(ctx, gcpProject)
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
func (c *Client) ReadImpactfulClusters(ctx context.Context, opts ImpactfulClusterReadOptions) ([]*ClusterSummary, error) {
	if opts.Thresholds == nil {
		return nil, errors.New("thresholds must be specified")
	}

	dataset, err := bqutil.DatasetForProject(opts.Project)
	if err != nil {
		return nil, errors.Annotate(err, "getting dataset").Err()
	}

	q := c.client.Query(`
		SELECT
			cluster_algorithm as ClusterAlgorithm,
			cluster_id as ClusterID,` +
		selectCounts("presubmit_rejects", "PresubmitRejects", "1d") +
		selectCounts("presubmit_rejects", "PresubmitRejects", "3d") +
		selectCounts("presubmit_rejects", "PresubmitRejects", "7d") +
		selectCounts("test_run_fails", "TestRunFails", "1d") +
		selectCounts("test_run_fails", "TestRunFails", "3d") +
		selectCounts("test_run_fails", "TestRunFails", "7d") +
		selectCounts("failures", "Failures", "1d") +
		selectCounts("failures", "Failures", "3d") +
		selectCounts("failures", "Failures", "7d") + `
			example_failure_reason.primary_error_message as ExampleFailureReason,
			example_test_id as ExampleTestID
		FROM ` + dataset + `.cluster_summaries
		WHERE (failures_1d > @unexpFailThreshold1d
			OR failures_3d > @unexpFailThreshold3d
			OR failures_7d > @unexpFailThreshold7d)
		ORDER BY
			failures_1d DESC,
			failures_3d DESC,
			failures_7d DESC
	`)
	q.Parameters = []bigquery.QueryParameter{
		{
			Name:  "unexpFailThreshold1d",
			Value: valueOrDefault(opts.Thresholds.UnexpectedFailures_1D, math.MaxInt64),
		},
		{
			Name:  "unexpFailThreshold3d",
			Value: valueOrDefault(opts.Thresholds.UnexpectedFailures_3D, math.MaxInt64),
		},
		{
			Name:  "unexpFailThreshold7d",
			Value: valueOrDefault(opts.Thresholds.UnexpectedFailures_7D, math.MaxInt64),
		},
	}
	job, err := q.Run(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "querying cluster summaries").Err()
	}
	it, err := job.Read(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "obtain result iterator").Err()
	}
	clusters := []*ClusterSummary{}
	for {
		row := &ClusterSummary{}
		err := it.Next(row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, errors.Annotate(err, "obtain next cluster summary row").Err()
		}
		clusters = append(clusters, row)
	}
	return clusters, nil
}

func valueOrDefault(value *int64, defaultValue int64) int64 {
	if value != nil {
		return *value
	}
	return defaultValue
}

// selectCounts generates SQL to select a set of Counts.
func selectCounts(sqlPrefix, fieldPrefix, suffix string) string {
	return `STRUCT(` +
		sqlPrefix + `_` + suffix + ` AS Nominal,` +
		sqlPrefix + `_pre_exon_` + suffix + ` AS PreExoneration,` +
		sqlPrefix + `_residual_` + suffix + ` AS Residual,` +
		sqlPrefix + `_residual_pre_exon_` + suffix + ` AS ResidualPreExoneration` +
		`) AS ` + fieldPrefix + suffix + `,`
}

// ReadCluster reads information about a single cluster.
func (c *Client) ReadCluster(ctx context.Context, luciProject, clusterAlgorithm, clusterID string) (*ClusterSummary, error) {
	dataset, err := bqutil.DatasetForProject(luciProject)
	if err != nil {
		return nil, errors.Annotate(err, "getting dataset").Err()
	}

	q := c.client.Query(`
		SELECT
			cluster_algorithm as ClusterAlgorithm,
			cluster_id as ClusterID,` +
		selectCounts("presubmit_rejects", "PresubmitRejects", "1d") +
		selectCounts("presubmit_rejects", "PresubmitRejects", "3d") +
		selectCounts("presubmit_rejects", "PresubmitRejects", "7d") +
		selectCounts("test_run_fails", "TestRunFails", "1d") +
		selectCounts("test_run_fails", "TestRunFails", "3d") +
		selectCounts("test_run_fails", "TestRunFails", "7d") +
		selectCounts("failures", "Failures", "1d") +
		selectCounts("failures", "Failures", "3d") +
		selectCounts("failures", "Failures", "7d") + `
			affected_tests_1d as AffectedTests1d,
			affected_tests_3d as AffectedTests3d,
			affected_tests_7d as AffectedTests7d,
			example_failure_reason.primary_error_message as ExampleFailureReason,
			example_test_id as ExampleTestID
		FROM ` + dataset + `.cluster_summaries
		WHERE cluster_algorithm = @clusterAlgorithm
		  AND cluster_id = @clusterID
	`)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "clusterAlgorithm", Value: clusterAlgorithm},
		{Name: "clusterID", Value: clusterID},
	}
	job, err := q.Run(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "querying cluster summary").Err()
	}
	it, err := job.Read(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "obtain result iterator").Err()
	}
	clusters := []*ClusterSummary{}
	for {
		row := &ClusterSummary{}
		err := it.Next(row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, errors.Annotate(err, "obtain next cluster summary row").Err()
		}
		clusters = append(clusters, row)
	}
	if len(clusters) == 0 {
		return nil, fmt.Errorf("cluster %s not found", clusterID)
	}
	return clusters[0], nil
}
