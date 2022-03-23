// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/spantest"
	"go.chromium.org/luci/server/span"
)

// cleanupDatabase deletes all data from all tables.
func cleanupDatabase(ctx context.Context, client *spanner.Client) error {
	_, err := client.Apply(ctx, []*spanner.Mutation{
		// No need to explicitly delete interleaved tables.
		spanner.Delete("AnalyzedTestVariants", spanner.AllKeys()),
		spanner.Delete("ClusteringState", spanner.AllKeys()),
		spanner.Delete("FailureAssociationRules", spanner.AllKeys()),
		spanner.Delete("Ingestions", spanner.AllKeys()),
		spanner.Delete("ReclusteringRuns", spanner.AllKeys()),
	})
	return err
}

// SpannerTestContext returns a context for testing code that talks to Spanner.
// Skips the test if integration tests are not enabled.
//
// Tests that use Spanner must not call t.Parallel().
func SpannerTestContext(tb testing.TB) context.Context {
	return spantest.SpannerTestContext(tb, cleanupDatabase)
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
	spantest.SpannerTestMain(m, findInitScript)
}

// MustApply applies the mutations to the spanner client in the context.
// Asserts that application succeeds.
// Returns the commit timestamp.
func MustApply(ctx context.Context, ms ...*spanner.Mutation) time.Time {
	ct, err := span.Apply(ctx, ms)
	So(err, ShouldBeNil)
	return ct
}
