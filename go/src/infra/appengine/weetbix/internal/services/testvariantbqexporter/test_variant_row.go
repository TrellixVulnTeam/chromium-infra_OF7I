// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testvariantbqexporter

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/googleapi"

	"go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/span"
	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/bqutil"
	bqpb "infra/appengine/weetbix/proto/bq"
)

func (b *BQExporter) query(ctx context.Context, f func(*bqpb.TestVariantRow) error) error {
	return fmt.Errorf("not implemented")
}

func (b *BQExporter) queryTestVariantsToExport(ctx context.Context, batchC chan []*bqpb.TestVariantRow) error {
	ctx, cancel := span.ReadOnlyTransaction(ctx)
	defer cancel()

	tvrs := make([]*bqpb.TestVariantRow, 0, maxBatchRowCount)
	rowCount := 0
	err := b.query(ctx, func(tvr *bqpb.TestVariantRow) error {
		tvrs = append(tvrs, tvr)
		rowCount++
		if len(tvrs) >= maxBatchRowCount {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case batchC <- tvrs:
			}
			tvrs = make([]*bqpb.TestVariantRow, 0, maxBatchRowCount)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(tvrs) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case batchC <- tvrs:
		}
	}

	logging.Infof(ctx, "fetched %d rows for exporting %s test variants", rowCount, b.options.Realm)
	return nil
}

func hasReason(apiErr *googleapi.Error, reason string) bool {
	for _, e := range apiErr.Errors {
		if e.Reason == reason {
			return true
		}
	}
	return false
}

func (b *BQExporter) batchExportRows(ctx context.Context, ins *bqutil.Inserter, batchC chan []*bqpb.TestVariantRow) error {
	eg, ctx := errgroup.WithContext(ctx)
	defer eg.Wait()

	for rows := range batchC {
		rows := rows
		if err := b.batchSem.Acquire(ctx, 1); err != nil {
			return err
		}

		eg.Go(func() error {
			defer b.batchSem.Release(1)
			err := b.insertRows(ctx, ins, rows)
			if bqutil.FatalError(err) {
				err = tq.Fatal.Apply(err)
			}
			return err
		})
	}

	return eg.Wait()
}

// insertRows inserts rows into BigQuery.
// Retries on transient errors.
func (b *BQExporter) insertRows(ctx context.Context, ins *bqutil.Inserter, rowProtos []*bqpb.TestVariantRow) error {
	if err := b.putLimiter.Wait(ctx); err != nil {
		return err
	}

	rows := make([]*bq.Row, 0, len(rowProtos))
	for _, ri := range rowProtos {
		row := &bq.Row{
			Message:  ri,
			InsertID: bigquery.NoDedupeID,
		}
		rows = append(rows, row)
	}

	return ins.PutWithRetries(ctx, rows)
}

func (b *BQExporter) exportTestVariantRows(ctx context.Context, ins *bqutil.Inserter) error {
	batchC := make(chan []*bqpb.TestVariantRow)
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return b.batchExportRows(ctx, ins, batchC)
	})

	eg.Go(func() error {
		defer close(batchC)
		return b.queryTestVariantsToExport(ctx, batchC)
	})

	return eg.Wait()
}
