// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reclustering

import (
	"context"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/analysis"
	"infra/appengine/weetbix/internal/analysis/clusteredfailures"
	"infra/appengine/weetbix/internal/clustering/chunkstore"
	"infra/appengine/weetbix/internal/clustering/reclustering"
	"infra/appengine/weetbix/internal/config"
	"infra/appengine/weetbix/internal/tasks/taskspb"
)

const (
	taskClass = "reclustering"
	queue     = "reclustering"
)

var tc = tq.RegisterTaskClass(tq.TaskClass{
	ID:        taskClass,
	Prototype: &taskspb.ReclusterChunks{},
	Queue:     queue,
	Kind:      tq.NonTransactional,
})

// RegisterTaskHandler registers the handler for reclustering tasks.
func RegisterTaskHandler(srv *server.Server) error {
	ctx := srv.Context
	cfg, err := config.Get(ctx)
	if err != nil {
		return err
	}
	chunkStore, err := chunkstore.NewClient(ctx, cfg.ChunkGcsBucket)
	if err != nil {
		return err
	}
	srv.RegisterCleanup(func(ctx context.Context) {
		chunkStore.Close()
	})
	cf := clusteredfailures.NewClient(srv.Options.CloudProject)
	analysis := analysis.NewClusteringHandler(cf)
	worker := reclustering.NewWorker(chunkStore, analysis)

	handler := func(ctx context.Context, payload proto.Message) error {
		task := payload.(*taskspb.ReclusterChunks)
		return reclusterTestResults(ctx, worker, task)
	}
	tc.AttachHandler(handler)
	return nil
}

// Schedule enqueues a task to recluster a range of chunks in a LUCI
// Project.
func Schedule(ctx context.Context, title string, task *taskspb.ReclusterChunks) error {
	return tq.AddTask(ctx, &tq.Task{
		Title: title,
		// Copy the task to avoid the caller retaining an alias to
		// the task proto passed to tq.AddTask.
		Payload: proto.Clone(task).(*taskspb.ReclusterChunks),
	})
}

func reclusterTestResults(ctx context.Context, worker *reclustering.Worker, task *taskspb.ReclusterChunks) error {
	err := worker.Do(ctx, task)
	if err != nil {
		logging.Errorf(ctx, "Error re-clustering: ", err)
		return err
	}
	return nil
}
