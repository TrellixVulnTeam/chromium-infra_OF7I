// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"cloud.google.com/go/bigquery"
	"github.com/maruel/subcommands"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
)

func cmdFetchRejections(authOpt *auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `fetch-rejections`,
		ShortDesc: "fetch test rejection data",
		LongDesc: text.Doc(`
			Fetch change rejection data, suitable for model creation.
			For format details, see comments of Rejection protobuf message.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &fetchRejectionsRun{}
			r.authOpt = authOpt
			r.RegisterBaseFlags(&r.Flags)
			r.Flags.IntVar(&r.minCLFlakes, "min-cl-flakes", 5, text.Doc(`
				In order to conlude that a test variant is flaky and exclude it from analysis,
				it must have mixed results in <min-cl-flakes> unique CLs.
			`))
			return r
		},
	}
}

type fetchRejectionsRun struct {
	baseCommandRun
	baseHistoryRun
	minCLFlakes int
}

func (r *fetchRejectionsRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)
	if len(args) != 0 {
		return r.done(errors.New("unexpected positional arguments"))
	}

	if err := r.baseHistoryRun.Init(ctx); err != nil {
		return r.done(err)
	}

	return r.done(r.runAndFetchResults(
		ctx,
		rejectedPatchSetsSQL,
		bigquery.QueryParameter{
			Name:  "minCLFlakes",
			Value: r.minCLFlakes,
		},
	))
}

// rejectedPatchSetsSQL is a BigQuery query that returns patchsets with test
// failures. Excludes flaky tests.
const rejectedPatchSetsSQL = commonSubqueries + `
	-- Test results annotated with patchset.
	ps_test_results AS (
		SELECT change, patchset, ps_approx_timestamp, tr.*
		FROM tryjobs t
		JOIN test_results_base tr ON t.id = tr.build_id
		WHERE not exonerated  AND status != 'SKIP' -- not needed for RTS purposes
	),

	-- The following two sub-queries detect flaky tests.
	-- It is important to exclude them from analysis because they represent
	-- noise.
	--
	-- A test variant is considered flaky if it has mixed results in >=N
	-- separate CLs. N=1 is too small because otherwise the query is vulnerable
	-- to a single bad patchset that introduces flakiness and never lands.
	--
	-- The first sub-query finds test variants with mixed results per CL.
	-- Note that GROUP BY includes patchset, but SELECT doesn't and uses
	-- DISTINCT.
	-- The second sub-query filters out test variant candidates that have mixed
	-- results win fewer than @minCLFlakes CLs.
	flaky_test_variants_per_cl AS (
		SELECT DISTINCT change, testVariant.id as test_id, variant_hash
		FROM ps_test_results
		GROUP BY change, patchset, test_id, variant_hash
		HAVING LOGICAL_OR(expected) AND LOGICAL_OR(NOT expected)
	),
	flaky_test_variants AS (
		SELECT test_id, variant_hash
		FROM flaky_test_variants_per_cl
		GROUP BY test_id, variant_hash
		HAVING COUNT(change) >= @minCLFlakes
	),

	-- Select test variants with unexpected results for each patchset.
	-- Does not filter out flaky test variants just yet.
	failed_test_variants_per_ps AS (
		SELECT
			change,
			patchset,
			variant_hash,
			MIN(ps_approx_timestamp) as ps_approx_timestamp,
			ANY_VALUE(testVariant) as testVariant
		FROM ps_test_results tr
		GROUP BY change, patchset, tr.testVariant.id, variant_hash
		HAVING LOGICAL_AND(NOT expected)
	)

-- Exclude flaky tests from failed_test_variants_per_ps.
-- Join all tables and produce rows in the Rejection protojson format.
SELECT
	patchsetArray(change, patchset, ANY_VALUE(af.files)) AS patchsets,
	RFC3339(MIN(ps_approx_timestamp)) as timestamp,
	ARRAY_AGG(testVariant) as failedTestVariants
FROM failed_test_variants_per_ps f
LEFT JOIN flaky_test_variants flaky ON f.testVariant.id = flaky.test_id AND f.variant_hash = flaky.variant_hash
-- flaky.test_id is NULL if LEFT JOIN did not find a flaky_test_variants row
-- for this test variant, i.e. the test is not flaky
JOIN affected_files af USING (change, patchset)
WHERE flaky.test_id IS NULL
GROUP BY change, patchset
`
