// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cloud.google.com/go/spanner"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/spantest"
	"go.chromium.org/luci/server/span"

	. "github.com/smartystreets/goconvey/convey"
)

const (
	// IntegrationTestEnvVar is the name of the environment variable which controls
	// whether spanner tests are executed.
	// The value must be "1" for integration tests to run.
	IntegrationTestEnvVar = "INTEGRATION_TESTS"
)

// runIntegrationTests returns true if integration tests should run.
func runIntegrationTests() bool {
	return os.Getenv(IntegrationTestEnvVar) == "1"
}

var spannerClient *spanner.Client

// SpannerTestContext returns a context for testing code that talks to Spanner.
// Skips the test if integration tests are not enabled.
//
// Tests that use Spanner must not call t.Parallel().
func SpannerTestContext(tb testing.TB) context.Context {
	switch {
	case !runIntegrationTests():
		tb.Skipf("env var %s=1 is missing", IntegrationTestEnvVar)
	case spannerClient == nil:
		tb.Fatalf("spanner client is not initialized; forgot to call SpannerTestMain?")
	}

	// Do not mock clock in integration tests because we cannot mock Spanner's
	// clock.
	ctx := testingContext(false)

	// All higher-level Convey scopes are run for each nested Convey call.
	// So spanner will try to apply the same mutations in the top-level for each
	// nested Convey call.
	// Clean up database to avoid the "AlreadyExists" errors.
	err := cleanupDatabase(ctx, spannerClient)
	if err != nil {
		tb.Fatal(err)
	}
	ctx = span.UseClient(ctx, spannerClient)
	return ctx
}

// findInitScript returns path //weetbix/internal/span/init_db.sql.
func findInitScript() (string, error) {
	ancestor, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}

	for {
		scriptPath := filepath.Join(ancestor, "internal", "span", "init_db.sql")
		_, err := os.Stat(scriptPath)
		if os.IsNotExist(err) {
			parent := filepath.Dir(ancestor)
			if parent == ancestor {
				return "", errors.Reason("init_db.sql not found").Err()
			}
			ancestor = parent
			continue
		}

		return scriptPath, err
	}
}

// SpannerTestMain is a test main function for packages that have tests that
// talk to spanner. It creates/destroys a temporary spanner database
// before/after running tests.
//
// This function never returns. Instead it calls os.Exit with the value returned
// by m.Run().
func SpannerTestMain(m *testing.M) {
	exitCode, err := spannerTestMain(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Exit(exitCode)
}

func spannerTestMain(m *testing.M) (exitCode int, err error) {
	testing.Init()

	if runIntegrationTests() {
		ctx := context.Background()
		start := clock.Now(ctx)
		var instanceName string
		var emulator *spantest.Emulator

		var err error
		// Start Cloud Spanner Emulator.
		if emulator, err = spantest.StartEmulator(ctx); err != nil {
			return 0, err
		}
		defer func() {
			switch stopErr := emulator.Stop(); {
			case stopErr == nil:

			case err == nil:
				err = stopErr

			default:
				fmt.Fprintf(os.Stderr, "failed to stop the emulator: %s\n", stopErr)
			}
		}()

		// Create a Spanner instance.
		if instanceName, err = emulator.NewInstance(ctx, ""); err != nil {
			return 0, err
		}
		fmt.Printf("started cloud emulator instance and created a temporary Spanner instance %s in %s\n", instanceName, time.Since(start))
		start = clock.Now(ctx)

		// Find init_db.sql
		initScriptPath, err := findInitScript()
		if err != nil {
			return 0, err
		}

		// Create a Spanner database.
		db, err := spantest.NewTempDB(ctx, spantest.TempDBConfig{InitScriptPath: initScriptPath, InstanceName: instanceName}, emulator)
		if err != nil {
			return 0, errors.Annotate(err, "failed to create a temporary Spanner database").Err()
		}
		fmt.Printf("created a temporary Spanner database %s in %s\n", db.Name, time.Since(start))

		defer func() {
			switch dropErr := db.Drop(ctx); {
			case dropErr == nil:

			case err == nil:
				err = dropErr

			default:
				fmt.Fprintf(os.Stderr, "failed to drop the database: %s\n", dropErr)
			}
		}()

		// Create a global Spanner client.
		spannerClient, err = db.Client(ctx)
		if err != nil {
			return 0, err
		}
	}

	return m.Run(), nil
}

// cleanupDatabase deletes all data from all tables.
func cleanupDatabase(ctx context.Context, client *spanner.Client) error {
	_, err := client.Apply(ctx, []*spanner.Mutation{
		// No need to explicitly delete interleaved tables.
		spanner.Delete("AnalyzedTestVariants", spanner.AllKeys()),
		spanner.Delete("BugClusters", spanner.AllKeys()),
		spanner.Delete("ClusteringState", spanner.AllKeys()),
	})
	return err
}

// MustApply applies the mutations to the spanner client in the context.
// Asserts that application succeeds.
// Returns the commit timestamp.
func MustApply(ctx context.Context, ms ...*spanner.Mutation) time.Time {
	ct, err := span.Apply(ctx, ms)
	So(err, ShouldBeNil)
	return ct
}
