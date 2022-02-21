// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ninjalog

import (
	"context"
	"fmt"
	"os"
	"testing"

	"cloud.google.com/go/bigquery"
)

func TestUpdateBQTable(t *testing.T) {
	if os.Getenv("ENABLE_NINJA_LOG_BQ_INTEGRATION_TEST") != "1" {
		t.Skipf("ENABLE_NINJA_LOG_BQ_INTEGRATION_TEST env var is not set")
	}

	const (
		stagingProjectID = "chromium-build-stats-staging"
		testTable        = "test"
	)

	t.Log("running integration test")
	ctx := context.Background()
	if err := CreateBQTable(ctx, stagingProjectID, testTable); err != nil {
		t.Fatalf("failed to run CreateBQTable with initialization: %v", err)
	}

	if err := UpdateBQTable(ctx, stagingProjectID, testTable); err != nil {
		t.Fatalf("failed to run UpdateBQTable without initilization: %v", err)
	}

	info := NinjaLog{
		Filename: ".ninja_log",
		Start:    1,
		Steps:    append([]Step{}, stepsTestCase...),
		Metadata: metadataTestCase,
	}

	gcsPath := "ninjalog_test_avro/test.avro"

	// Upload test ninja log to test bucket in avro format.
	if err := WriteNinjaLogToGCS(ctx, &info, stagingProjectID+".appspot.com", gcsPath); err != nil {
		t.Fatalf("failed to write ninja log to GCS: %v", err)
	}

	client, err := bigquery.NewClient(ctx, stagingProjectID)
	if err != nil {
		t.Fatalf("failed to create BigQuery client: %v", err)
	}

	// Load uploaded ninja log to test BQ table from GCS.
	gcsRef := bigquery.NewGCSReference(fmt.Sprintf("gs://%s.appspot.com/%s", stagingProjectID, gcsPath))
	gcsRef.SourceFormat = bigquery.Avro
	gcsRef.AvroOptions = &bigquery.AvroOptions{UseAvroLogicalTypes: true}

	loader := client.Dataset("ninjalog").Table(testTable).LoaderFrom(gcsRef)

	// Do not update existing schema.
	loader.CreateDisposition = bigquery.CreateNever
	loader.WriteDisposition = bigquery.WriteAppend

	job, err := loader.Run(ctx)
	if err != nil {
		t.Fatalf("failed to start load job: %v", err)
	}

	status, err := job.Wait(ctx)
	if err != nil {
		t.Fatalf("failed to wait load job: %v", err)
	}

	if err := status.Err(); err != nil {
		t.Fatalf("failed to finish job successfully: %v", err)
	}
}
