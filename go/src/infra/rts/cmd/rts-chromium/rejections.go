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
	-- Group all test results by patchset, test_id and variant_hash
	-- in order to analyze individual test variants in each patchset,
	-- and in particular exclude flaky tests.
	test_variants_per_ps AS (
		SELECT
			change,
			patchset,
			testVariant.id AS test_id,
			variant_hash,
			ANY_VALUE(testVariant) as testVariant,
			LOGICAL_OR(expected) AND LOGICAL_OR(NOT expected) AS flake,
			LOGICAL_AND(NOT expected) all_unexpected,
			ANY_VALUE(ps_approx_timestamp) AS ps_approx_timestamp,
		FROM tryjobs t
		JOIN test_results_base tr ON t.id = tr.build_id
		WHERE not exonerated  AND status != 'SKIP'  -- not needed for RTS purposes
		GROUP BY change, patchset, test_id, variant_hash

		# Exclude all-expected results early on.
		HAVING NOT LOGICAL_AND(expected)
	),

	-- Find all true test failures, without flakes.
	failed_test_variants AS (
		SELECT
			test_id,
			variant_hash,
			ANY_VALUE(testVariant) AS testVariant,
			ARRAY_AGG(
				IF(all_unexpected, STRUCT(change, patchset, ps_approx_timestamp), NULL)
				IGNORE NULLS
			) AS patchsets_with_failures,
		FROM test_variants_per_ps
		GROUP BY test_id, variant_hash
		# Exclude test variants where flakiness was observed in @minCLFlakes CLs
		# (not patchsets) or more.
		HAVING COUNT(DISTINCT IF(flake, change, NULL)) < @minCLFlakes
	)

-- Join all tables and produce rows in the Rejection protojson format.
SELECT
	patchsetArray(change, patchset, ANY_VALUE(files)) AS patchsets,
	RFC3339(MIN(ps_approx_timestamp)) as timestamp,
	ARRAY_AGG(testVariant) as failedTestVariants
FROM failed_test_variants tv, tv.patchsets_with_failures
JOIN affected_files USING (change, patchset)
GROUP BY change, patchset
`
