// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package state

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"infra/appengine/weetbix/internal/clustering"
	cpb "infra/appengine/weetbix/internal/clustering/proto"
	"infra/appengine/weetbix/internal/config"
	spanutil "infra/appengine/weetbix/internal/span"

	"cloud.google.com/go/spanner"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/server/span"
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
	// Clustering describes the latest clustering of test results in
	// the chunk.
	Clustering clustering.ClusterResults
	// LastUpdated is the Spanner commit time the row was last updated. Output only.
	LastUpdated time.Time
}

// NotFound is the error returned by Read if the row could not be found.
var NotFound = errors.New("clustering state row not found")

// EndOfTable is the highest possible chunk ID that can be stored.
var EndOfTable = strings.Repeat("ff", 16)

// Create inserts clustering state for a chunk. Must be
// called in the context of a Spanner transaction.
func Create(ctx context.Context, e *Entry) error {
	if err := validateEntry(e); err != nil {
		return err
	}
	clusters, err := encodeClusters(e.Clustering.Algorithms, e.Clustering.Clusters)
	if err != nil {
		return err
	}
	ms := spanutil.InsertMap("ClusteringState", map[string]interface{}{
		"Project":           e.Project,
		"ChunkID":           e.ChunkID,
		"PartitionTime":     e.PartitionTime,
		"ObjectID":          e.ObjectID,
		"AlgorithmsVersion": e.Clustering.AlgorithmsVersion,
		"RulesVersion":      e.Clustering.RulesVersion,
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
	whereClause := "ChunkID = @chunkID"
	params := make(map[string]interface{})
	params["chunkID"] = chunkID

	limit := 1
	results, err := readWhere(ctx, project, whereClause, params, limit)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		// Row does not exist.
		return nil, NotFound
	}
	return results[0], nil
}

// ReadNextOptions specifies options for ReadNextN.
type ReadNextOptions struct {
	// The exclusive lower bound of the range of ChunkIDs to read.
	// To read from the start of the table, leave this blank ("").
	StartChunkID string
	// The inclusive upper bound of the range of ChunkIDs to read.
	// To specify the end of the table, use the constant EndOfTable.
	EndChunkID string
	// The minimum AlgorithmsVersion that re-clustering wants to achieve.
	// If a row has an AlgorithmsVersion less than this value, it will
	// be eligble to be read.
	AlgorithmsVersion int64
	// The minimum RulesVersion that re-clustering wants to achieve.
	// If a row has an RulesVersion less than this value, it will
	// be eligble to be read.
	RulesVersion time.Time
}

// ReadNextN reads the n consecutively next clustering state entries
// matching ReadNextOptions.
func ReadNextN(ctx context.Context, project string, opts ReadNextOptions, n int) ([]*Entry, error) {
	params := make(map[string]interface{})
	whereClause := `
		ChunkId > @startChunkID AND ChunkId <= @endChunkID
		AND (AlgorithmsVersion < @algorithmsVersion OR RulesVersion < @rulesVersion)
	`
	params["startChunkID"] = opts.StartChunkID
	params["endChunkID"] = opts.EndChunkID
	params["algorithmsVersion"] = opts.AlgorithmsVersion
	params["rulesVersion"] = opts.RulesVersion

	return readWhere(ctx, project, whereClause, params, n)
}

func readWhere(ctx context.Context, project, whereClause string, params map[string]interface{}, limit int) ([]*Entry, error) {
	stmt := spanner.NewStatement(`
		SELECT
		  ChunkId, PartitionTime, ObjectId,
		  AlgorithmsVersion, RulesVersion,
		  LastUpdated, Clusters
		FROM ClusteringState
		WHERE Project = @project AND (` + whereClause + `)
		ORDER BY ChunkId
		LIMIT @limit
	`)
	for k, v := range params {
		stmt.Params[k] = v
	}
	stmt.Params["project"] = project
	stmt.Params["limit"] = limit

	it := span.Query(ctx, stmt)
	var b spanutil.Buffer
	results := []*Entry{}
	err := it.Do(func(r *spanner.Row) error {
		clusters := &cpb.ChunkClusters{}
		result := &Entry{Project: project}

		err := b.FromSpanner(r,
			&result.ChunkID, &result.PartitionTime, &result.ObjectID,
			&result.Clustering.AlgorithmsVersion, &result.Clustering.RulesVersion,
			&result.LastUpdated, clusters)
		if err != nil {
			return errors.Annotate(err, "read clustering state row").Err()
		}
		result.Clustering.Algorithms, result.Clustering.Clusters, err = decodeClusters(clusters)
		if err != nil {
			return errors.Annotate(err, "decode clusters").Err()
		}
		results = append(results, result)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// EstimateChunks estimates the total number of chunks in the ClusteringState
// table for the given project.
func EstimateChunks(ctx context.Context, project string) (int, error) {
	stmt := spanner.NewStatement(`
	  SELECT ChunkId
	  FROM ClusteringState
	  WHERE Project = @project
	  ORDER BY ChunkId ASC
	  LIMIT 1 OFFSET 100
	`)
	stmt.Params["project"] = project

	it := span.Query(ctx, stmt)
	var chunkID string
	err := it.Do(func(r *spanner.Row) error {
		if err := r.Columns(&chunkID); err != nil {
			return errors.Annotate(err, "read ChunkID row").Err()
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	if chunkID == "" {
		// There was no 100th chunk ID. There must be less
		// than 100 chunks in the project.
		return 99, nil
	}
	return estimateChunksFromID(chunkID)
}

// estimateChunksFromID estimates the number of chunks in a project
// given the ID of the 100th chunk (in ascending keyspace order) in
// that project. The maximum estimate that will be returned is one
// billion. If there is no 100th chunk ID in the project, then
// there are clearly 99 chunks or less in the project.
func estimateChunksFromID(chunkID100 string) (int, error) {
	const MaxEstimate = 1000 * 1000 * 1000
	// This function uses the property that ChunkIDs are approximately
	// uniformly distributed. We use the following estimator of the
	// number of rows:
	//   100 / (fraction of keyspace used up to 100th row)
	// where fraction of keyspace used up to 100th row is:
	//   (ChunkID_100th + 1) / 2^128
	//
	// Where ChunkID_100th is the ChunkID of the 100th row (in keyspace
	// order), as a 128-bit integer (rather than hexadecimal string).
	//
	// Rearranging this estimator, we get:
	//   100 * 2^128 / (ChunkID_100th + 1)

	// numerator = 100 * 2 ^ 128
	var numerator big.Int
	numerator.Lsh(big.NewInt(100), 128)

	idBytes, err := hex.DecodeString(chunkID100)
	if err != nil {
		return 0, err
	}

	// denominator = ChunkID_100th + 1. We add one because
	// the keyspace consumed includes the ID itself.
	var denominator big.Int
	denominator.SetBytes(idBytes)
	denominator.Add(&denominator, big.NewInt(1))

	// estimate = numerator / denominator.
	var estimate big.Int
	estimate.Div(&numerator, &denominator)

	result := uint64(math.MaxUint64)
	if estimate.IsUint64() {
		result = estimate.Uint64()
	}
	if result > MaxEstimate {
		result = MaxEstimate
	}
	return int(result), nil
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
	case e.Clustering.AlgorithmsVersion <= 0:
		return errors.New("algorithms version must be specified")
	case e.Clustering.RulesVersion.IsZero():
		return errors.New("rules version must be specified")
	default:
		if err := validateAlgorithms(e.Clustering.Algorithms); err != nil {
			return errors.Annotate(err, "algorithms").Err()
		}
		if err := validateClusters(e.Clustering.Clusters); err != nil {
			return errors.Annotate(err, "clusters").Err()
		}
		return nil
	}
}

func validateAlgorithms(algorithms map[string]struct{}) error {
	for a := range algorithms {
		if !clustering.AlgorithmRe.MatchString(a) {
			return fmt.Errorf("algorithm %q is not valid", a)
		}
	}
	return nil
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
