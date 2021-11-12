// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testvariantbqexporter

import (
	"context"
	"fmt"
	"net/url"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/pbutil"
	pb "infra/appengine/weetbix/proto/v1"
)

const (
	taskClass = "export-test-variants"
	queue     = "export-test-variants"
)

// RegisterTaskClass registers the task class for tq dispatcher.
func RegisterTaskClass() {
	tq.RegisterTaskClass(tq.TaskClass{
		ID:        taskClass,
		Prototype: &taskspb.ExportTestVariants{},
		Queue:     queue,
		Kind:      tq.NonTransactional,
		Handler: func(ctx context.Context, payload proto.Message) error {
			task := payload.(*taskspb.ExportTestVariants)
			br := CreateBQExporter(&Options{
				Realm:        task.Realm,
				CloudProject: task.CloudProject,
				Dataset:      task.Dataset,
				Table:        task.Table,
				Predicate:    task.Predicate,
				TimeRange:    task.TimeRange,
			})
			return br.ExportRows(ctx)
		},
	})
}

// Schedule enqueues a task to export AnalyzedTestVariant rows to BigQuery.
func Schedule(ctx context.Context, realm, cloudProject, dataset, table string, predicate *pb.AnalyzedTestVariantPredicate, timeRange *pb.TimeRange) error {
	earliest, err := pbutil.AsTime(timeRange.Earliest)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s-%s-%s-%s-%d", realm, cloudProject, dataset, url.PathEscape(table), earliest.Unix())
	return tq.AddTask(ctx, &tq.Task{
		Title: key,
		Payload: &taskspb.ExportTestVariants{
			Realm:        realm,
			CloudProject: cloudProject,
			Dataset:      dataset,
			Table:        table,
			Predicate:    predicate,
			TimeRange:    timeRange,
		},
		DeduplicationKey: key,
	})
}
