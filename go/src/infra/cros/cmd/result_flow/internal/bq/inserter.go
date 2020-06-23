// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bq

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/time/rate"
	bqapi "google.golang.org/api/bigquery/v2"
	"google.golang.org/api/googleapi"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/retry"
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/common/sync/dispatcher"
	"go.chromium.org/luci/common/sync/dispatcher/buffer"
)

// Original RAMBufferedBQInserter is at "infra/qscheduler/service/app/eventlog/ram.go".

// Inserter implements an interface to interact with Bigquery.
// TODO: consider moving the interface to the consumer side.
type Inserter interface {
	Insert(context.Context, ...bigquery.ValueSaver) error
	Close()
	CloseAndDrain(context.Context)
}

// Options presents the options for the interface to interact with Bigquery.
type Options struct {
	Target        *result_flow.BigqueryConfig
	HTTPClient    *http.Client
	InsertRPCMock func(context.Context, *bqapi.TableDataInsertAllRequest) (*bqapi.TableDataInsertAllResponse, error)
}

// ramBufferedBQInserter implements AsyncBqInserter via in-RAM buffering of events
// for later sending to BQ.
type ramBufferedBQInserter struct {
	ProjectID string
	DatasetID string
	TableID   string

	httpClient *http.Client

	// channel does the heavy lifting for us:
	//  * buffering,
	//  * back-pressure,
	//  * ratelimiting,
	//  * retries,
	// allowing ramBufferedBQInserter to focus on sending the data.
	channel dispatcher.Channel

	// insertRPCMock is used by tests to mock actual BQ insert API call.
	insertRPCMock func(context.Context, *bqapi.TableDataInsertAllRequest) (*bqapi.TableDataInsertAllResponse, error)
}

// NewInserter instantiates new Inserter.
func NewInserter(ctx context.Context, op Options) (Inserter, error) {
	const (
		qps          = 10.0
		burst        = 15
		maxLeases    = 10
		batchSize    = 100  // 100 BQ rows sent at once.
		maxLiveItems = 1000 // At most these many items not yet currently leased for sending.
	)
	var err error
	r := &ramBufferedBQInserter{
		ProjectID:  op.Target.Project,
		DatasetID:  op.Target.Dataset,
		TableID:    op.Target.Table,
		httpClient: op.HTTPClient,
	}
	r.channel, err = dispatcher.NewChannel(
		ctx,
		&dispatcher.Options{
			QPSLimit: rate.NewLimiter(qps, burst),
			Buffer: buffer.Options{
				MaxLeases: maxLeases,
				BatchSize: batchSize,
				FullBehavior: &buffer.DropOldestBatch{
					MaxLiveItems: maxLiveItems,
				},
				Retry: inserterRetry,
			},
		},
		func(batch *buffer.Batch) error { return r.send(ctx, batch) },
	)
	r.insertRPCMock = op.InsertRPCMock
	return r, err
}

// Insert inserts rows to BQ asynchronously.
func (r *ramBufferedBQInserter) Insert(ctx context.Context, rows ...bigquery.ValueSaver) error {
	for _, row := range rows {
		rowMap, insertID, err := row.Save()
		if err != nil {
			return errors.Annotate(err, "failed to get row map from %s", row).Err()
		}
		rowJSON, err := valuesToJSON(rowMap)
		if err != nil {
			return errors.Annotate(err, "failed to JSON-serialize BQ row %s", row).Err()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r.channel.C <- &bqapi.TableDataInsertAllRequestRows{InsertId: insertID, Json: rowJSON}:
		}
	}
	return nil
}

// CloseAndDrain stops accepting new rows and waits until all buffered rows are
// sent or provided `ctx` times out.
func (r *ramBufferedBQInserter) CloseAndDrain(ctx context.Context) {
	r.channel.CloseAndDrain(ctx)
}

// Close closes the Inserter and swallows the panic if already closed.
func (r *ramBufferedBQInserter) Close() {
	r.channel.Close()
}

func (r *ramBufferedBQInserter) send(ctx context.Context, batch *buffer.Batch) error {
	rows := make([]*bqapi.TableDataInsertAllRequestRows, 0, len(batch.Data))
	for _, d := range batch.Data {
		rows = append(rows, d.(*bqapi.TableDataInsertAllRequestRows)) // despite '...Rows', it's just 1 row.
	}

	logging.Infof(ctx, "Sending TestPlanRun %s", rows[0].InsertId)
	f := r.insertRPC
	if r.insertRPCMock != nil {
		f = r.insertRPCMock
	}
	// NOTE: dispatcher.Channel retries for us if error is transient.
	resp, err := f(ctx, &bqapi.TableDataInsertAllRequest{
		SkipInvalidRows: true, // they will be reported in lastResp.InsertErrors
		Rows:            rows,
	})
	if err != nil {
		if isTransientError(err) {
			err = transient.Tag.Apply(err)
		}
		return errors.Annotate(err, "sending to BigQuery").Err()
	}

	if len(resp.InsertErrors) > 0 {
		// Use only first error as a sample. Dumping them all is impractical.
		blob, _ := json.MarshalIndent(resp.InsertErrors[0].Errors, "", "  ")
		logging.Errorf(ctx, "%d rows weren't accepted, sample error:\n%s", len(resp.InsertErrors), blob)
	}
	return nil
}

// insertRPC does the actual BigQuery insert.
//
// It is mocked in tests.
func (r *ramBufferedBQInserter) insertRPC(ctx context.Context, req *bqapi.TableDataInsertAllRequest) (
	*bqapi.TableDataInsertAllResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	bq, err := bqapi.New(r.httpClient)
	if err != nil {
		return nil, err
	}
	call := bq.Tabledata.InsertAll(r.ProjectID, r.DatasetID, r.TableID, req)
	return call.Context(ctx).Do()
}

// valuesToJSON prepares row map in a format used by BQ API.
func valuesToJSON(in map[string]bigquery.Value) (map[string]bqapi.JsonValue, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make(map[string]bqapi.JsonValue, len(in))
	for k, v := range in {
		blob, err := json.Marshal(v)
		if err != nil {
			return nil, errors.Annotate(err, "failed to JSON-serialize key %q", k).Err()
		}
		out[k] = json.RawMessage(blob)
	}
	return out, nil
}

func inserterRetry() retry.Iterator {
	return &retry.ExponentialBackoff{
		Limited: retry.Limited{
			Delay:    50 * time.Millisecond,
			Retries:  50,
			MaxTotal: 45 * time.Second,
		},
		Multiplier: 2,
	}
}

func isTransientError(e error) bool {
	if gerr, _ := e.(*googleapi.Error); gerr != nil {
		if gerr.Code >= 500 {
			return true
		}
	}
	return false
}
