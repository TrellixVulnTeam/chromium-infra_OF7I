// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/maruel/subcommands"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
)

func cmdFetchDurations(authOpt *auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `fetch-durations`,
		ShortDesc: "fetch test duration data",
		LongDesc: text.Doc(`
			Fetch test duration data, suitable for model creation.
			For format details, see comments of TestDurationRecord protobuf message.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &fetchDurationsRun{}
			r.authOpt = authOpt
			r.RegisterBaseFlags(&r.Flags)
			r.Flags.Float64Var(&r.frac, "frac", 0.1, "Fraction of the data to fetch")
			r.Flags.DurationVar(&r.minDuration, "min-duration", time.Second, "Minimum duration to fetch")
			return r
		},
	}
}

type fetchDurationsRun struct {
	baseCommandRun
	baseHistoryRun
	frac        float64
	minDuration time.Duration
}

func (r *fetchDurationsRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)
	if len(args) != 0 {
		return r.done(errors.New("unexpected positional arguments"))
	}

	if err := r.baseHistoryRun.Init(ctx); err != nil {
		return r.done(err)
	}

	return r.done(r.runAndFetchResults(
		ctx,
		testDurationsSQL,
		bigquery.QueryParameter{
			Name:  "frac",
			Value: r.frac,
		},
		bigquery.QueryParameter{
			Name:  "minDuration",
			Value: r.minDuration.Seconds(),
		},
	))
}

const commonSubqueries = `
# Returns an array of structs compatible with GerritPatch protojson message.
CREATE TEMP FUNCTION patchsetArray(change INT64, patchset INT64, files ARRAY<STRING>) AS (
  [STRUCT(
		STRUCT(
			"chromium-review.googlesource.com" AS host,
			"chromium/src" AS project,
			change AS number
		) AS change,
		patchset,
		ARRAY(
			SELECT AS STRUCT
				"https://chromium.googlesource.com/chromium/src" AS repo,
				f as path
			FROM UNNEST(files) f
		) AS changedFiles
	)]
);

CREATE TEMP FUNCTION RFC3339(ts TIMESTAMP)
RETURNS STRING
LANGUAGE js
AS "return ts.toISOString()";

WITH
	affected_files_raw AS (
		SELECT
			ps.change,
			ps.patchset,
			-- Extract affected files from the property
			-- and replace "src/" prefix with "//".
			ARRAY(
					SELECT REGEXP_REPLACE(JSON_EXTRACT_SCALAR(file_name_json, "$"), "^src/", "//")
					FROM UNNEST(JSON_EXTRACT_ARRAY(b.output.properties, "$.affected_files.first_100")) file_name_json
			) AS files,
		FROM cr-buildbucket.chromium.builds b, b.input.gerrit_changes ps
		WHERE create_time BETWEEN @startTime and TIMESTAMP_ADD(@endTime, INTERVAL 1 DAY)
			-- Skip patchsets that modified >100 files, because we don't have the full
			-- list. This also skips builds that don't have affected flies, e.g. CI builds,
			-- because the value is NULL.
			AND CAST(JSON_EXTRACT(b.output.properties, "$.affected_files.total_count") as FLOAT64) <= 100

			-- Ignore any builds that modified non-src.git files.
			AND NOT EXISTS (
				SELECT 0
				FROM UNNEST(JSON_EXTRACT_ARRAY(b.output.properties, "$.affected_files.first_100")) f
				-- The leading quote is there because it is a JSON string.
				WHERE f NOT LIKE '"src/%'
			)
	),

	affected_files AS (
		-- Choose the longest file list.
		-- File lists for the same patchset can be different if the parent CL landed
		-- between bot_updates of different tryjobs.
		SELECT change, patchset, ARRAY_AGG(af ORDER BY ARRAY_LENGTH(files) DESC LIMIT 1)[OFFSET(0)].files
		FROM affected_files_raw af
		GROUP BY change, patchset
	),

	tryjobs AS (
		SELECT
			b.id,
			ps.change,
			ps.earliest_equivalent_patchset as patchset,
			partition_time as ps_approx_timestamp,
		FROM commit-queue.chromium.attempts a, a.gerrit_changes ps, a.builds b
		WHERE partition_time BETWEEN @startTime AND @endTime
	),

	test_results_base AS (
		SELECT
			CAST(REGEXP_EXTRACT(exported.id, r'build-(\d+)') as INT64) as build_id,
			# This struct corresponds to TestVariant protojson message.
			STRUCT(
				test_id AS id,
				ARRAY(SELECT FORMAT("%s:%s", key, value) kv FROM UNNEST(variant) ORDER BY kv) as variant,
				test_metadata.location.file_name as fileName
			) as testVariant,
			variant_hash,
			expected,
			exonerated,
			status,
			duration,
		FROM luci-resultdb.chromium.try_test_results tr
		-- Read prev-day and next-day results too to ensure that we have ALL
		-- results of a given CQ attempt.
		WHERE partition_time BETWEEN TIMESTAMP_SUB(@startTime, INTERVAL 1 DAY) and TIMESTAMP_ADD(@endTime, INTERVAL 1 DAY)
			AND (@test_id_regexp = '' OR REGEXP_CONTAINS(test_id, @test_id_regexp))
			AND (@builder_regexp = '' OR EXISTS (SELECT 0 FROM tr.variant WHERE key='builder' AND REGEXP_CONTAINS(value, @builder_regexp)))

			# Exclude third-party tests (except Web Tests) because they test code
			# which isn't in src.git.
			# As of January 2021, this excludes ~2% of test results.
			AND (
				test_metadata.location.file_name NOT LIKE '%/third_party/%'
				OR test_metadata.location.file_name LIKE '//third_party/blink/%'
			)
	),
`

const testDurationsSQL = commonSubqueries + `
	test_results AS (
		SELECT *
		FROM test_results_base
		WHERE RAND() <= @frac
			AND duration > @minDuration

			# Exclude tests which don't have a file name.
			# This means the prediction is less representative of the build-level
			# savings, but on the other hand it excludes the noise created by
			# test frameworks that don't report file names yet (i.e. things that the
			# candidate strategy doesn't have control over), thus providing a stronger
			# signal for the strategy.
			AND testVariant.fileName != ''
	)

-- Join all tables and produce rows in the TestDurationRecord protojson format.
SELECT
	patchsetArray(change, patchset, ANY_VALUE(af.files)) AS patchsets,
	ARRAY_AGG(STRUCT(testVariant, FORMAT("%fs", duration) as duration)) AS testDurations,
FROM tryjobs t
JOIN test_results tr ON t.id = tr.build_id
JOIN affected_files af USING (change, patchset)
GROUP BY
	# Produce separate TestDurationRecords for different builders,
	# otherwise we hit per-row BigQuery limits.
	# This doesn't affect computation results. It just means batches are split
	# by builder.
	(SELECT v FROM tr.testVariant.variant v WHERE v LIKE 'builder:%'),
	change, patchset
`
