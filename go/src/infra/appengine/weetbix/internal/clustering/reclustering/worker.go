// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reclustering

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	cpb "infra/appengine/weetbix/internal/clustering/proto"
	"infra/appengine/weetbix/internal/clustering/runs"
	"infra/appengine/weetbix/internal/clustering/state"
	"infra/appengine/weetbix/internal/tasks/taskspb"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/retry"
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/server/span"
)

// ChunkStore is the interface for the blob store archiving chunks of test
// results for later re-clustering.
type ChunkStore interface {
	// Get retrieves the chunk with the specified object ID and returns it.
	Get(ctx context.Context, project, objectID string) (*cpb.Chunk, error)
}

// Worker provides methods to process re-clustering tasks. It is safe to be
// used by multiple threads concurrently.
type Worker struct {
	chunkStore ChunkStore
	analysis   Analysis
}

// NewWorker initialises a new Worker.
func NewWorker(chunkStore ChunkStore, analysis Analysis) *Worker {
	return &Worker{
		chunkStore: chunkStore,
		analysis:   analysis,
	}
}

// taskContext provides objects relevant to working on a particular
// re-clustering task.
type taskContext struct {
	worker        *Worker
	task          *taskspb.ReclusterChunks
	run           *runs.ReclusteringRun
	progressToken *runs.ProgressToken
	// The exclusive lower bound defining the range of ChunkIds
	// still to cluster.
	currentChunkID string
}

// Do handles a re-clustering task.
func (w *Worker) Do(ctx context.Context, task *taskspb.ReclusterChunks) error {
	attemptTime := task.AttemptTime.AsTime()

	run, err := runs.Read(span.Single(ctx), task.Project, attemptTime)
	if err != nil {
		return errors.Annotate(err, "read run for task").Err()
	}

	pt := runs.NewProgressToken(task.Project, attemptTime)

	tctx := &taskContext{
		worker:         w,
		task:           task,
		run:            run,
		progressToken:  pt,
		currentChunkID: task.StartChunkId,
	}

	// attemptTime is the (soft) deadline for the run.
	for clock.Now(ctx).Before(attemptTime) {
		var done bool
		err := retry.Retry(ctx, transient.Only(retry.Default), func() error {
			var err error
			done, err = tctx.recluster(ctx)
			return err
		}, nil)
		if err != nil {
			return err
		}
		if done {
			break
		}
	}
	return nil
}

// recluster tries to reclusters some chunks, advancing currentChunkID
// as it succeeds. It returns 'true' if all chunks to be re-clustered by
// the reclustering task were completed.
func (t *taskContext) recluster(ctx context.Context) (done bool, err error) {
	const batchSize = 10
	readOpts := state.ReadNextOptions{
		StartChunkID:      t.currentChunkID,
		EndChunkID:        t.task.EndChunkId,
		AlgorithmsVersion: t.run.AlgorithmsVersion,
		RulesVersion:      t.run.RulesVersion,
	}
	entries, err := state.ReadNextN(span.Single(ctx), t.task.Project, readOpts, batchSize)
	if err != nil {
		return false, errors.Annotate(err, "read next chunk state").Err()
	}
	if len(entries) == 0 {
		// We have finished re-clustering.
		err = t.progressToken.ReportProgress(ctx, 1000)
		if err != nil {
			return true, errors.Annotate(err, "report progress").Err()
		}
		return true, nil
	}
	for _, entry := range entries {
		progress, err := calculateProgress(t.task, entry.ChunkID)
		if err != nil {
			return false, errors.Annotate(err, "calculate progress").Err()
		}

		err = t.progressToken.ReportProgress(ctx, progress)
		if err != nil {
			return false, errors.Annotate(err, "report progress").Err()
		}

		// Read the test results from GCS.
		chunk, err := t.worker.chunkStore.Get(ctx, t.task.Project, entry.ObjectID)
		if err != nil {
			return false, errors.Annotate(err, "read chunk").Err()
		}

		// Re-cluster the test results in spanner, then export
		// the re-clustering to BigQuery for analysis.
		err = Update(ctx, t.worker.analysis, chunk, entry)
		if err != nil {
			if err == state.UpdateRaceErr {
				// Our update raced with another update (or a delete).
				// This is retriable if we re-read the chunk again.
				err = transient.Tag.Apply(err)
			}
			return false, errors.Annotate(err, "re-cluster chunk").Err()
		}

		// Advance our position.
		t.currentChunkID = entry.ChunkID
	}

	// More to do.
	return false, nil
}

// calculateProgress calculates the progress of the worker through the task.
// Progress is the proportion of the keyspace re-clustered, as a value between
// 0 and 1000 (i.e. 0 = 0%, 1000 = 100.0%).
// 1000 is never returned by this method as the value passed is the nextChunkID
// (i.e. the next chunkID to re-cluster), not the last completed chunk ID,
// which implies progress is not complete.
func calculateProgress(task *taskspb.ReclusterChunks, nextChunkID string) (int, error) {
	nextID, err := chunkIDAsBigInt(nextChunkID)
	if err != nil {
		return 0, err
	}
	startID, err := chunkIDAsBigInt(task.StartChunkId)
	if err != nil {
		return 0, err
	}
	endID, err := chunkIDAsBigInt(task.EndChunkId)
	if err != nil {
		return 0, err
	}
	if startID.Cmp(endID) >= 0 {
		return 0, fmt.Errorf("end chunk ID %q is before or equal to start %q", task.EndChunkId, task.StartChunkId)
	}
	if nextID.Cmp(startID) <= 0 {
		// Start is exclusive, not inclusive.
		return 0, fmt.Errorf("next chunk ID %q is before or equal to start %q", nextChunkID, task.StartChunkId)
	}
	if nextID.Cmp(endID) > 0 {
		return 0, fmt.Errorf("next chunk ID %q is after end %q", nextChunkID, task.EndChunkId)
	}

	// progress = (((nextID - 1) - startID) * 1000) / (endID - startID)
	var numerator big.Int
	numerator.Sub(nextID, big.NewInt(1))
	numerator.Sub(&numerator, startID)
	numerator.Mul(&numerator, big.NewInt(1000))

	var denominator big.Int
	denominator.Sub(endID, startID)

	var result big.Int
	result.Div(&numerator, &denominator)

	return int(result.Uint64()), nil
}

// chunkIDAsBigInt represents a 128-bit chunk ID
// (normally represented as 32 lowercase hexadecimal characters)
// as a big.Int.
func chunkIDAsBigInt(chunkID string) (*big.Int, error) {
	if chunkID == "" {
		// "" indicates start of table. This is one before
		// ID 00000 .... 00000.
		return big.NewInt(-1), nil
	}
	idBytes, err := hex.DecodeString(chunkID)
	if err != nil {
		return nil, err
	}
	id := big.NewInt(0)
	id.SetBytes(idBytes)
	return id, nil
}
