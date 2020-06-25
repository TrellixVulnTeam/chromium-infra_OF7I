// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bq_test

import (
	"context"
	"fmt"
	"infra/cros/cmd/result_flow/internal/bq"
	"sync"
	"testing"

	"cloud.google.com/go/bigquery"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	bqapi "google.golang.org/api/bigquery/v2"
)

// Original TestRAMBufferedBQInserter was at "infra/qscheduler/service/app/eventlog/ram_test.go".
// Tests here verify that insert_id is propagated correctly because it is important for deduplication.

func TestRamBufferedBQInserter(t *testing.T) {
	fakeConfig := &result_flow.BigqueryConfig{
		Project: "test-project",
		Dataset: "test-dataset",
		Table:   "test-table",
	}

	Convey("With mock context", t, func() {
		ctx := context.Background()
		ctx = logging.SetLevel(gologger.StdConfig.Use(ctx), logging.Debug)
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		Convey("verify insertIDs", func() {
			var lock sync.Mutex
			insertIDs := stringset.Set{}
			fakeOptions := bq.Options{
				Target:     fakeConfig,
				HTTPClient: nil,
				InsertRPCMock: func(_ context.Context, req *bqapi.TableDataInsertAllRequest) (*bqapi.TableDataInsertAllResponse, error) {
					lock.Lock()
					for _, row := range req.Rows {
						insertIDs.Add(row.InsertId)
					}
					lock.Unlock()
					return &bqapi.TableDataInsertAllResponse{}, nil
				},
			}
			entries := make([]bigquery.ValueSaver, 5)
			for i := 0; i < 5; i++ {
				entries[i] = mkTestEntry(fmt.Sprintf("given:%d", i))
			}
			bi, _ := bq.NewInserter(ctx, fakeOptions)
			So(bi.Insert(ctx, entries...), ShouldBeNil)
			bi.CloseAndDrain(ctx)
			// All rows should have non-empty insert_id.
			So(insertIDs.Has(""), ShouldBeFalse)
			// insert_id should be recognized.
			So(insertIDs.HasAll("given:0", "given:1", "given:2", "given:3", "given:4"), ShouldBeTrue)
			So(insertIDs.Len(), ShouldEqual, 5)
		})
	})
}

func mkTestEntry(insertID string) bigquery.ValueSaver {
	return &testEntry{InsertID: insertID, Data: map[string]bigquery.Value{
		"uid": insertID,
	}}
}

type testEntry struct {
	InsertID string
	Data     map[string]bigquery.Value
}

// Save implements bigquery.ValueSaver interface.
func (e testEntry) Save() (map[string]bigquery.Value, string, error) {
	return e.Data, e.InsertID, nil
}
