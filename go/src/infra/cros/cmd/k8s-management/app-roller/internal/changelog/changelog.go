// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package changelog implements a K8s resource change logging on contexts.
package changelog

import (
	"context"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"
)

// WithBQUploader returns a context that can be used to log changes to a
// BigQuery table.
func WithBQUploader(ctx context.Context, project, dataset, table, serviceAccountJSON string) (context.Context, func()) {
	authCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	bqc, err := bigquery.NewClient(authCtx, project, option.WithCredentialsFile(serviceAccountJSON))
	if err != nil {
		log.Printf("chagnelog: Will skip BigQuery uploading: %s", err)
		return ctx, func() {}
	}

	r := &changeRecord{}
	uploadFunc := func() {
		r.upload(ctx, bqc.Dataset(dataset).Table(table).Inserter())
	}
	ctx = context.WithValue(ctx, key{}, r)
	return ctx, uploadFunc

}

// LogChange logs the change for BigQuery uploading.
func LogChange(ctx context.Context, c interface{}) {
	r, ok := ctx.Value(key{}).(*changeRecord)
	if !ok {
		return
	}
	r.add(c)
}

// key is a context value key.
type key struct{}

// changeRecord is a thread-safe struct to save all records for uploading.
type changeRecord struct {
	// recordsMu protects 'records'.
	recordsMu sync.Mutex
	records   []interface{}
}

// add adds a record for uploading.
func (u *changeRecord) add(r interface{}) {
	u.recordsMu.Lock()
	defer u.recordsMu.Unlock()
	u.records = append(u.records, r)
}

// upload uploads records to BigQuery.
func (u *changeRecord) upload(ctx context.Context, inserter *bigquery.Inserter) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := inserter.Put(ctx, u.records); err != nil {
		log.Printf("changelog: put %d record(s) to BigQuery failed: %s", len(u.records), err)
	} else {
		log.Printf("changelog: put %d record(s) to BigQuery", len(u.records))
	}
}
