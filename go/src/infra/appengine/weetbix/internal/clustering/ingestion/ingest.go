// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ingestion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"go.chromium.org/luci/common/errors"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server/span"

	cpb "infra/appengine/weetbix/internal/clustering/proto"
	"infra/appengine/weetbix/internal/clustering/reclustering"
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/clustering/state"
	"infra/appengine/weetbix/internal/config"
	"infra/appengine/weetbix/internal/config/compiledcfg"
	pb "infra/appengine/weetbix/proto/v1"
)

// Options represents parameters to the ingestion.
type Options struct {
	// Project is the LUCI Project.
	Project string
	// PartitionTime is the start of the retention period of test results
	// being ingested.
	PartitionTime time.Time
	// Realm is the LUCI Realm of the test results.
	Realm string
	// InvocationID is the identity of the invocation being ingested.
	InvocationID string
	// ImplicitlyExonerateBlockingFailures controls whether invocation-blocking
	// failures should be automatically treated as exonerated, regardless of
	// exoneration status reported to ResultDB.
	// This is set if either:
	// - the build corresponding to the ingested invocation was cancelled,
	//   passed, or had an infra failure (i.e. was anything other than a
	//   build failure), or
	// - the CQ run did not consider the build critical (e.g. because it
	//   was experimental).
	// As inferences that test failures caused the release build or CQ run
	// to fail are self-evidently untrue in those cases.
	ImplicitlyExonerateBlockingFailures bool
	// PresubmitRunID is the identity of the presubmit run (if any).
	PresubmitRunID *pb.PresubmitRunId
	// PresubmitRunOwner is the the owner of the presubmit
	// run (if any). This is the owner of the CL on which CQ+1/CQ+2 was
	// clicked (even in case of presubmit run with multiple CLs).
	PresubmitRunOwner string
	// PresubmitRunCls are the Changelists included in the presubmit run
	// (if any). Changelists must be sorted in ascending
	// (host, change, patchset) order. Up to 10 changelists may be captured.
	PresubmitRunCls []*pb.Changelist
}

// ChunkStore is the interface for the blob store archiving chunks of test
// results for later re-clustering.
type ChunkStore interface {
	// Put saves the given chunk to storage. If successful, it returns
	// the randomly-assigned ID of the created object.
	Put(ctx context.Context, project string, content *cpb.Chunk) (string, error)
}

// ChunkSize is the number of test failures that are to appear in each chunk.
const ChunkSize = 1000

// Ingester handles the ingestion of test results for clustering.
type Ingester struct {
	chunkStore ChunkStore
	analysis   reclustering.Analysis
}

// New initialises a new Ingester.
func New(cs ChunkStore, a reclustering.Analysis) *Ingester {
	return &Ingester{
		chunkStore: cs,
		analysis:   a,
	}
}

// Ingestion handles the ingestion of a single invocation for clustering,
// in a streaming fashion.
type Ingestion struct {
	// ingestor provides access to shared objects for doing the ingestion.
	ingester *Ingester
	// opts is the Ingestion options.
	opts Options
	// buffer is the set of failures which have been queued for ingestion but
	// not yet written to chunks.
	buffer []*cpb.Failure
	// page is the number of the next page of failures to be written out.
	page int
}

// Open commences the ingestion of a new invocation, with the specified
// options.
func (i *Ingester) Open(opts Options) *Ingestion {
	return &Ingestion{
		ingester: i,
		opts:     opts,
	}
}

// Put buffers test results for clustering. They will be periodically written
// out as chunks. Once all test results have been buffered, call Flush to
// ensure all chunks are written out.
func (i *Ingestion) Put(ctx context.Context, tvs []*rdbpb.TestVariant) error {
	failures := failuresFromTestVariants(i.opts, tvs)
	i.buffer = append(i.buffer, failures...)

	for len(i.buffer) > ChunkSize {
		chunk := i.takeChunk()
		if err := i.writeChunk(ctx, chunk); err != nil {
			return err
		}
	}
	return nil
}

// Flush clears the ingestion buffer and writes out the last chunk (if any).
// This should only be called once all calls to Put() have been made, otherwise
// it introduces non-determinism in the chunks test failures appear in, which
// breaks retry behaviour.
func (i *Ingestion) Flush(ctx context.Context) error {
	chunk := i.takeChunk()
	if chunk == nil {
		// All test failures have been written already.
		return nil
	}
	if err := i.writeChunk(ctx, chunk); err != nil {
		return err
	}
	return nil
}

func (i *Ingestion) takeChunk() *cpb.Chunk {
	end := ChunkSize
	if end > len(i.buffer) {
		end = len(i.buffer)
	}
	if end == 0 {
		// No chunk to take.
		return nil
	}

	// Copy the data, to avoid keeping the taken chunk's failures
	// in memory once they have been written out.
	// https://go.dev/blog/slices-intro
	page := make([]*cpb.Failure, end)
	copy(page, i.buffer[0:end])

	newBuffer := make([]*cpb.Failure, len(i.buffer)-end)
	copy(newBuffer, i.buffer[end:len(i.buffer)])
	i.buffer = newBuffer

	for i, f := range page {
		f.ChunkIndex = int64(i)
	}
	return &cpb.Chunk{
		Failures: page,
	}
}

// writeChunk will, for the given chunk:
// - Archive the failures to GCS.
// - Cluster the failures.
// - Write out the chunk clustering state.
// - Perform analysis.
func (i *Ingestion) writeChunk(ctx context.Context, chunk *cpb.Chunk) error {
	// Derive a chunkID deterministically from the ingested root invocation
	// ID and page number. In case of retry this avoids ingesting the same
	// data twice.
	id := chunkID(i.opts.InvocationID, i.page)
	i.page++

	_, err := state.Read(span.Single(ctx), i.opts.Project, id)
	if err == nil {
		// Chunk was already ingested as part of an earlier ingestion attempt.
		// Do not attempt to ingest again.
		return nil
	}
	if err != state.NotFoundErr {
		return err
	}

	// Upload the chunk. The objectID is randomly generated each time
	// so the actual insertion of the chunk will be atomic with the
	// ClusteringState row in Spanner.
	objectID, err := i.ingester.chunkStore.Put(ctx, i.opts.Project, chunk)
	if err != nil {
		return err
	}

	clusterState := &state.Entry{
		Project:       i.opts.Project,
		ChunkID:       id,
		PartitionTime: i.opts.PartitionTime,
		ObjectID:      objectID,
	}

	ruleset, err := reclustering.Ruleset(ctx, i.opts.Project, rules.StartingEpoch)
	if err != nil {
		return errors.Annotate(err, "obtain ruleset").Err()
	}

	cfg, err := compiledcfg.Project(ctx, i.opts.Project, config.StartingEpoch)
	if err != nil {
		return errors.Annotate(err, "obtain config").Err()
	}

	update, err := reclustering.PrepareUpdate(ctx, ruleset, cfg, chunk, clusterState)
	if err != nil {
		return err
	}

	updates := reclustering.NewPendingUpdates(ctx)
	updates.Add(update)
	if err := updates.Apply(ctx, i.ingester.analysis); err != nil {
		return err
	}
	return nil
}

// chunkID generates an identifier for the chunk deterministically.
// The identifier will be 32 lowercase hexadecimal characters. Generated
// identifiers will be approximately evenly distributed through
// the keyspace.
func chunkID(rootInvocationID string, seq int) string {
	content := fmt.Sprintf("%q:%v", rootInvocationID, seq)
	sha256 := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sha256[:16])
}
