// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package bquploader defines a uploader of BigQuery.
package bquploader

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"
)

// Uploader uploads records to BigQuery at regular intervals.
type Uploader struct {
	closing   chan struct{}
	ticker    *time.Ticker
	wg        sync.WaitGroup
	inserter  *bigquery.Inserter
	recordsMu sync.Mutex // recordsMu protects 'records'.
	records   []interface{}
}

// TargetTable represents a fully qualified BigQuery table name.
type TargetTable struct {
	ProjectID string
	Dataset   string
	TableName string
}

func (t *TargetTable) String() string {
	return fmt.Sprintf("%s.%s.%s", t.ProjectID, t.Dataset, t.TableName)
}

// NewUploader creates a new Uploader.
func NewUploader(table TargetTable, interval time.Duration, opts ...option.ClientOption) (*Uploader, error) {
	authCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	bq, err := bigquery.NewClient(authCtx, table.ProjectID, opts...)
	if err != nil {
		return nil, err
	}
	r := bq.Dataset(table.Dataset).Table(table.TableName).Inserter()
	log.Printf("bquploader: created for %s", table.String())

	u := &Uploader{
		closing:  make(chan struct{}),
		inserter: r,
		ticker:   time.NewTicker(interval),
		records:  make([]interface{}, 0, 1000),
	}
	u.wg.Add(1)
	go func() {
		defer u.wg.Done()
		for {
			select {
			case <-u.ticker.C:
				u.upload()
			case <-u.closing:
				// Upload the remaining records before close.
				u.upload()
				return
			}
		}
	}()
	return u, nil
}

// QueueRecord queues a record for uploading in the next batch.
func (u *Uploader) QueueRecord(r interface{}) {
	u.recordsMu.Lock()
	defer u.recordsMu.Unlock()
	u.records = append(u.records, r)
}

// Close closes the uploader and release all resources.
func (u *Uploader) Close() {
	close(u.closing)
	u.ticker.Stop()
	u.wg.Wait()
}

func (u *Uploader) upload() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	u.recordsMu.Lock()
	toUpload := u.records
	u.records = make([]interface{}, 0, 1000)
	u.recordsMu.Unlock()

	if err := u.inserter.Put(ctx, toUpload); err != nil {
		log.Printf("bquploader: put %d record(s) to BigQuery failed: %s", len(toUpload), err)
	} else {
		log.Printf("bquploader: put %d record(s) to BigQuery", len(toUpload))
	}
}
