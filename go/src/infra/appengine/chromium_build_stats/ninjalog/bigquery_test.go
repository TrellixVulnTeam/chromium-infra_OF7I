// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ninjalog

import (
	"context"
	"os"
	"testing"
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
}
