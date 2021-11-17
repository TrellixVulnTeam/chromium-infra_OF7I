// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testvariantbqexporter

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/server/auth/realms"
	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/config"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/pbutil"
	pb "infra/appengine/weetbix/proto/v1"
)

const (
	taskClass = "export-test-variants"
	queue     = "export-test-variants"
	// BqExportJobInterval is the interval between two bq export cron jobs.
	// It's also used as the default time range of each row.
	BqExportJobInterval = time.Hour
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

// ScheduleTasks schedules tasks to export test variants to BigQuery.
// It schedules a task per realm per table.
func ScheduleTasks(ctx context.Context) error {
	pjcs, err := config.Projects(ctx)
	if err != nil {
		return errors.Annotate(err, "get project configs").Err()
	}

	// The cron job is scheduled to run at 0:00, 1:00 ..., and to export rows
	// containing data of the past hour.
	// In case this is a retry, round the time back to the full hour.
	latest := clock.Now(ctx).UTC().Truncate(time.Hour)
	if err != nil {
		return err
	}
	timeRange := &pb.TimeRange{
		Earliest: timestamppb.New(latest.Add(-BqExportJobInterval)),
		Latest:   timestamppb.New(latest),
	}

	var errs []error
	for pj, cg := range pjcs {
		for _, rc := range cg.GetRealms() {
			fullRealm := realms.Join(pj, rc.Name)
			bqcs := rc.GetTestVariantAnalysis().GetBqExports()
			for _, bqc := range bqcs {
				table := bqc.GetTable()
				if table == nil {
					continue
				}
				err := Schedule(ctx, fullRealm, table.CloudProject, table.Dataset, table.Table, bqc.GetPredicate(), timeRange)
				if err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
	if len(errs) > 0 {
		return errors.NewMultiError(errs...)
	}
	return nil
}
