// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package state

import (
	"context"
	"fmt"
	"time"

	"infra/appengine/weetbix/internal/clustering"
	cpb "infra/appengine/weetbix/internal/clustering/proto"
	"infra/appengine/weetbix/internal/config"
	spanutil "infra/appengine/weetbix/internal/span"

	"cloud.google.com/go/spanner"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/server/span"
	"google.golang.org/grpc/codes"
)

// Entry represents the clustering state of a chunk, consisting of:
// - Metadata about what test results were clustered.
// - Metadata about how the test results were clustered (the algorithms
//   and failure association rules used).
// - The clusters each test result are in.
type Entry struct {
	// Project is the LUCI Project the chunk belongs to.
	Project string
	// ChunkID is the identity of the chunk of test results. 32 lowercase hexadecimal
	// characters assigned by the ingestion process.
	ChunkID string
	// PartitionTime is the start of the retention period of the test results in the chunk.
	PartitionTime time.Time
	// ObjectID is the identity of the object in GCS containing the chunk's test results.
	// 32 lowercase hexadecimal characters.
	ObjectID string
	// AlgorithmsVersion is the version of clustering algorithms used to cluster test
	// results in this chunk. (This is a version over the set of algorithms, distinct
	// from the versions of a single algorithm, e.g.: v1 -> {reason-0.1}, v2 -> {reason-0.1,
	// testname-0.1}, v3 -> {reason-0.2, testname-0.1}.)
	AlgorithmsVersion int64
	// RuleVersion is the version of the set of failure association rules
	// used to match test results in this chunk. This is the RulesLastUpdated
	// time of the most-recently-updated failure association rule in the snapshot
	// of failure association rules used to match the test results.
	RuleVersion time.Time
	// Clusters records the clusters each test result in the cluster belongs to.
	Clusters [][]*clustering.ClusterID
	// LastUpdated is the Spanner commit time the row was last updated. Output only.
	LastUpdated time.Time
}

// NotFound is the error returned by Read if the row could not be found.
var NotFound = errors.New("clustering state row not found")

// Create inserts clustering state for a chunk. Must be
// called in the context of a Spanner transaction.
func Create(ctx context.Context, e *Entry) error {
	if err := validateEntry(e); err != nil {
		return err
	}
	clusters, err := encodeClusters(e.Clusters)
	if err != nil {
		return err
	}
	ms := spanutil.InsertMap("ClusteringState", map[string]interface{}{
		"Project":           e.Project,
		"ChunkID":           e.ChunkID,
		"PartitionTime":     e.PartitionTime,
		"ObjectID":          e.ObjectID,
		"AlgorithmsVersion": e.AlgorithmsVersion,
		"RuleVersion":       e.RuleVersion,
		"Clusters":          clusters,
		"LastUpdated":       spanner.CommitTimestamp,
	})
	span.BufferWrite(ctx, ms)
	return nil
}

// Read reads clustering state for a chunk. Must be
// called in the context of a Spanner transaction. If no clustering
// state exists, the method returns the error NotFound.
func Read(ctx context.Context, project, chunkID string) (*Entry, error) {
	columns := []string{
		"Project", "ChunkId", "PartitionTime",
		"ObjectId", "AlgorithmsVersion", "RuleVersion",
		"Clusters", "LastUpdated",
	}
	row, err := span.ReadRow(ctx, "ClusteringState", spanner.Key{project, chunkID}, columns)
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			// Row does not exist.
			return nil, NotFound
		}
		return nil, err
	}
	var b spanutil.Buffer
	result := &Entry{}
	clusters := &cpb.ChunkClusters{}
	err = b.FromSpanner(row,
		&result.Project, &result.ChunkID, &result.PartitionTime,
		&result.ObjectID, &result.AlgorithmsVersion, &result.RuleVersion,
		clusters, &result.LastUpdated)
	if err != nil {
		return nil, err
	}
	result.Clusters, err = decodeClusters(clusters)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func validateEntry(e *Entry) error {
	switch {
	case !config.ProjectRe.MatchString(e.Project):
		return fmt.Errorf("project %q is not valid", e.Project)
	case !clustering.ChunkRe.MatchString(e.ChunkID):
		return fmt.Errorf("chunk ID %q is not valid", e.ChunkID)
	case e.PartitionTime.IsZero():
		return errors.New("partition time must be specified")
	case e.ObjectID == "":
		return errors.New("object ID must be specified")
	case e.AlgorithmsVersion <= 0:
		return errors.New("algorithms version must be specified")
	case e.RuleVersion.IsZero():
		return errors.New("rule version must be specified")
	default:
		if err := validateClusters(e.Clusters); err != nil {
			return errors.Annotate(err, "clusters").Err()
		}
		return nil
	}
}

func validateClusters(clusters [][]*clustering.ClusterID) error {
	if len(clusters) == 0 {
		// Each chunk must have at least one test result, even
		// if that test result is in no clusters.
		return errors.New("there must be clustered test results in the chunk")
	}
	// Outer slice has on entry per test result.
	for i, tr := range clusters {
		// Inner slice has the list of clusters per test result.
		for j, c := range tr {
			if err := c.Validate(); err != nil {
				return errors.Annotate(err, "test result %v: cluster %v: cluster ID is not valid", i, j).Err()
			}
		}
	}
	return nil
}
