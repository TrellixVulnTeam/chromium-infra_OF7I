// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bufio"
	"context"
	"fmt"
	"infra/rts"
	"infra/rts/filegraph/git"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/maruel/subcommands"
	luciflag "go.chromium.org/luci/common/flag"
	"google.golang.org/api/iterator"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"

	"go.chromium.org/luci/common/cli"
)

const csvHeader = "ResultId,ChangeId,TestName,TestId,FileName,VariantHash,SixMonthRunCount,SixMonthFailCount,OneMonthRunCount,OneMonthFailCount,OneWeekRunCount,OneWeekFailCount,Distance,UseDistance,Failed"

func cmdGenTrainingData(authOpt *auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `gen-training-data -out <path> -day <date> -builder <builder>`,
		ShortDesc: "Generate features and labels for ML model",
		LongDesc: text.Doc(`
			Generate features and labels for ML model

			Flags -day -out -model-dir are required.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &genTraingData{}
			r.authOpt = authOpt
			r.Flags.StringVar(&r.modelDir, "model-dir", "", text.Doc(`
				Path to the directory with the model files.
				Normally it is coming from CIPD package "chromium/rts/model"
				and precomputed by "rts-chromium create-model" command.
			`))
			r.Flags.StringVar(&r.out, "out", "", text.Doc(`
				Filename to write csv training data to. If it already exists
				the file will be appended to.
			`))
			r.Flags.StringVar(&r.testSuite, "test-suite", "", text.Doc(`
				Test suite to get training data for.
			`))
			r.Flags.StringVar(&r.builder, "builder", "", text.Doc(`
				Builder to get training data for.
			`))
			r.Flags.Var(luciflag.Date(&r.runDate), "day", text.Doc(`
				Fetch results for this date. Stability information will be
				gathered based on this day.
				format: yyyy-mm-dd
			`))
			r.Flags.IntVar(&r.queryLimit, "query-limit", 1000000, text.Doc(`
				Max rows to get per query. This allows us to get around resource
				limits
			`))
			r.Flags.IntVar(&r.downSample, "down-sample", 1000, text.Doc(`
				The factor to down sample passes by to increase the number of
				failures. A value less than or equal to 0 will result in no
				down sampling
			`))
			r.Flags.BoolVar(&r.ignorePassedBuilds, "ignore-passed-jobs", false,
				"Whether or not to ignore results from builds that passed")
			r.Flags.BoolVar(&r.onlyTestFailures, "only-test-failures", false, text.Doc(`
				"Only return failure entries (intended for creating test sets to
				compare to RTS framework)
			`))
			return r
		},
	}
}

func (r *genTraingData) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := r.ValidateFlags(); err != nil {
		return r.done(err)
	}

	ctx := cli.GetContext(a, r, env)

	r.loadInput(ctx)
	r.openCsv()
	defer r.closeCsv()

	bqClient, err := newBQClient(ctx, auth.NewAuthenticator(ctx, auth.InteractiveLogin, *r.authOpt))
	if err != nil {
		return r.done(errors.Annotate(err, "failed to create BigQuery client").Err())
	}

	rows, err := r.queryResults(ctx, bqClient)
	if err != nil {
		return r.done(errors.Annotate(err, "failed to retrieve query results").Err())
	}

	fmt.Printf("Calculating distances\n")
	// Eventually we might want to restructure the query to group on CL for us.
	// The row limit makes this less straight forward and would require limiting
	// the agg like:
	// ARRAY_AGG(testVariant LIMIT @failedVariantsLimit) as failedTestVariants
	for len(rows) > 0 {
		r.calcDistancesForRows(rows)

		// If received the full buffer last time, query again
		if len(rows) == r.queryLimit {
			rows, err = r.queryResults(ctx, bqClient)
			if err != nil {
				return r.done(errors.Annotate(err, "failed to retrieve query results").Err())
			}
		}
	}
	return 0
}

func (r *genTraingData) calcDistancesForRows(rows []bqRow) {
	lastChangeId := rows[0].ChangeId
	var changeIdTestIdsToDistance = make(map[string]bqRow)
	for i := 0; i < len(rows); i++ {
		row := rows[i]
		if row.ChangeId != lastChangeId {
			// Find distances for the last patchset
			r.calcDistances(row.AffectedFiles, changeIdTestIdsToDistance)
			r.writeCsvRows(changeIdTestIdsToDistance)

			lastChangeId = row.ChangeId
			changeIdTestIdsToDistance = make(map[string]bqRow)
			r.currentClCount += 1
			if r.currentClCount%100 == 0 {
				fmt.Printf("Processed %d patchsets\n", r.currentClCount)
			}
		}
		changeIdTestIdsToDistance[row.FileName] = row
	}
	r.calcDistances(rows[len(rows)-1].AffectedFiles, changeIdTestIdsToDistance)
	r.writeCsvRows(changeIdTestIdsToDistance)
}

func (r *genTraingData) closeCsv() {
	r.file.Close()
}

// Opens the file if there was an input or creates the file if an output was
// provided
func (r *genTraingData) openCsv() error {
	// TODO(sshrimp): use encoding/csv instead
	if _, err := os.Stat(r.out); os.IsNotExist(err) {
		r.file, err = os.Create(r.out)
		if err != nil {
			return err
		}

		fmt.Fprintf(r.file, "%s\n", csvHeader)
	} else {
		// TODO(sshrimp): Check the header still matches
		r.file, err = os.OpenFile(r.out, os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *genTraingData) writeCsvRows(tests map[string]bqRow) {
	for _, row := range tests {
		s.writeCsv(row)
	}
}

func (r *genTraingData) writeCsv(row bqRow) {
	fmt.Fprintf(r.file, "%v,", strings.ReplaceAll(row.ResultId, ",", ";"))
	fmt.Fprintf(r.file, "%v,", strings.ReplaceAll(row.ChangeId, ",", ";"))
	fmt.Fprintf(r.file, "%v,", strings.ReplaceAll(row.TestName, ",", ";"))
	fmt.Fprintf(r.file, "%v,", strings.ReplaceAll(row.TestId, ",", ";"))
	fmt.Fprintf(r.file, "%v,", strings.ReplaceAll(row.FileName, ",", ";"))
	fmt.Fprintf(r.file, "%v,", strings.ReplaceAll(row.VariantHash, ",", ";"))
	fmt.Fprintf(r.file, "%v,", row.SixMonthRunCount)
	fmt.Fprintf(r.file, "%v,", row.SixMonthFailCount)
	fmt.Fprintf(r.file, "%v,", row.OneMonthRunCount)
	fmt.Fprintf(r.file, "%v,", row.OneMonthFailCount)
	fmt.Fprintf(r.file, "%v,", row.OneWeekRunCount)
	fmt.Fprintf(r.file, "%v,", row.OneWeekFailCount)

	// Training will handle missing values. This lets us continuously append
	// without having to keep track of any running values.
	if row.UseDistance {
		fmt.Fprintf(r.file, "%f,", row.Distance)
	} else {
		fmt.Fprintf(r.file, ",")
	}

	fmt.Fprintf(r.file, "%v,", row.UseDistance)
	fmt.Fprintf(r.file, "%v", row.Failed)
	fmt.Fprintf(r.file, "\n")
}

func (r *genTraingData) ValidateFlags() error {
	switch {
	case r.out == "":
		return errors.New("-out is required")
	case r.runDate.IsZero():
		return errors.New("-day is required")
	case r.modelDir == "":
		return errors.New("the -model-dir is required")
	default:
		return nil
	}
}

func (r *genTraingData) queryResults(ctx context.Context, bqClient *bigquery.Client) ([]bqRow, error) {
	q := bqClient.Query(filtersQuery)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "runDate", Value: r.runDate},
		{Name: "testSuite", Value: r.testSuite},
		{Name: "builder", Value: r.builder},
		{Name: "queryLimit", Value: r.queryLimit},
		{Name: "limitOffset", Value: r.queryIndex},
		{Name: "downSample", Value: r.downSample},
		{Name: "ignorePassedBuilds", Value: r.ignorePassedBuilds},
		{Name: "onlyTestFailures", Value: r.onlyTestFailures},
	}

	fmt.Printf("Querying for entries\n")
	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}

	rows := []bqRow{}
	row := &bqRow{}
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		// Read the next row.
		switch err := it.Next(row); {
		case err == iterator.Done:
			r.queryIndex += int64(len(rows))
			return rows, ctx.Err()
		case err != nil:
			return nil, err
		}

		rows = append(rows, *row)
	}
}

// loadInput loads all the input of the subcommand.
func (r *genTraingData) loadInput(ctx context.Context) error {
	fmt.Printf("Loading model\n")
	gitGraphDir := filepath.Join(r.modelDir, "git-file-graph")
	return r.loadGraph(filepath.Join(gitGraphDir, "graph.fg"))
}

func (r *genTraingData) loadGraph(fileName string) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	r.strategy.Graph = &git.Graph{}
	return r.strategy.Graph.Read(bufio.NewReader(f))
}

func (s *genTraingData) calcDistances(changedFiles []string, tests map[string]bqRow) {
	if len(changedFiles) > 100 {
		for _, row := range tests {
			row.UseDistance = false
		}
		return
	}

	foundTests := 0
	s.strategy.RunQuery(changedFiles, func(name string, af rts.Affectedness) (keepGoing bool) {
		if entry, ok := tests[name]; ok {
			entry.Distance = af.Distance
			entry.UseDistance = true
			tests[name] = entry
			foundTests += 1

			if len(tests) == foundTests {
				return false
			}
		}
		return true
	})
}

type genTraingData struct {
	baseCommandRun

	runDate            time.Time
	testSuite          string
	builder            string
	downSample         int
	ignorePassedBuilds bool
	onlyTestFailures   bool

	out            string
	modelDir       string
	strategy       git.SelectionStrategy
	file           *os.File
	queryLimit     int
	queryIndex     int64
	currentClCount int

	authOpt       *auth.Options
	authenticator *auth.Authenticator
	http          *http.Client
}

type bqRow struct {
	ResultId           string   `bigquery:"result_id"`
	ChangeId           string   `bigquery:"change_id"`
	TestName           string   `bigquery:"test_name"`
	TestId             string   `bigquery:"test_id"`
	FileName           string   `bigquery:"file_name"`
	VariantHash        string   `bigquery:"variant_hash"`
	SixMonthFailCount  int64    `bigquery:"six_month_fail_count"`
	SixMonthRunCount   int64    `bigquery:"six_month_run_count"`
	OneMonthFailCount  int64    `bigquery:"one_month_fail_count"`
	OneMonthRunCount   int64    `bigquery:"one_month_run_count"`
	OneWeekFailCount   int64    `bigquery:"one_week_fail_count"`
	OneWeekRunCount    int64    `bigquery:"one_week_run_count"`
	AffectedFilesCount int64    `bigquery:"affected_files_count"`
	AffectedFiles      []string `bigquery:"affected_files"`
	Failed             bool     `bigquery:"failed"`
	Distance           float64
	// Feature for if the distance couldn't be found, might want to split into
	// "InfiniteDistance" and "HasDistance" to better capture new tests
	UseDistance bool
}

const filtersQuery = `
WITH fail_rate as (
	SELECT
		ds.test_id test_id,
		ds.variant_hash variant_hash,
		SUM(ARRAY_LENGTH(ds.patchsets_with_failures)) six_month_fail_count,
		SUM(ds.run_count) six_month_run_count,
		SUM(IF(day >= TIMESTAMP_SUB(TIMESTAMP_TRUNC(@runDate, DAY), INTERVAL 31 DAY), ARRAY_LENGTH(ds.patchsets_with_failures), 0)) one_month_fail_count,
		SUM(IF(day >= TIMESTAMP_SUB(TIMESTAMP_TRUNC(@runDate, DAY), INTERVAL 31 DAY), ds.run_count, 0)) one_month_run_count,
		SUM(IF(day >= TIMESTAMP_SUB(TIMESTAMP_TRUNC(@runDate, DAY), INTERVAL 8 DAY), ARRAY_LENGTH(ds.patchsets_with_failures), 0)) one_week_fail_count,
		SUM(IF(day >= TIMESTAMP_SUB(TIMESTAMP_TRUNC(@runDate, DAY), INTERVAL 8 DAY), ds.run_count, 0)) one_week_run_count,
	FROM
		chrome-trooper-analytics.test_results.daily_summary ds
	WHERE
		ds.day BETWEEN
			TIMESTAMP_SUB(TIMESTAMP_TRUNC(@runDate, DAY), INTERVAL 181 DAY) AND
			TIMESTAMP_SUB(TIMESTAMP_TRUNC(@runDate, DAY), INTERVAL 1 DAY)
	GROUP BY ds.test_id, ds.variant_hash
),

tryjobs AS (
	SELECT
		TIMESTAMP_TRUNC(partition_time, DAY) day,
		b.id,
		ps.change,
		ps.earliest_equivalent_patchset as patchset,
		partition_time as ps_approx_timestamp,
	FROM commit-queue.chromium.attempts a, a.gerrit_changes ps, a.builds b
	WHERE
		TIMESTAMP_TRUNC(partition_time, DAY) = TIMESTAMP_TRUNC(@runDate, DAY)
),

bb_tryjobs AS (
	SELECT
		id,
		status,
		IF(JSON_EXTRACT(b.output.properties, "$.affected_files.total_count") IS NULL, 0, CAST(CAST(JSON_EXTRACT(b.output.properties, "$.affected_files.total_count") AS FLOAT64) AS INT)) affected_files_count,
		ARRAY(SELECT REGEXP_REPLACE(REPLACE(file, '"', ""), r'^src/', '//') FROM UNNEST(JSON_EXTRACT_ARRAY(b.output.properties, "$.affected_files.first_100")) file) affected_files
	FROM cr-buildbucket.chromium.builds b
	WHERE create_time BETWEEN TIMESTAMP_SUB(@runDate, INTERVAL 1 DAY) AND TIMESTAMP_ADD(@runDate, INTERVAL 1 DAY)
		AND builder.bucket = 'try'
		# Exclude experimental builders because they may fail for reasons
		# unrelated to the CL, and are not required for the CL to land.
		AND STRUCT('cq_experimental', 'true') NOT IN UNNEST(b.tags)
		AND (not @ignorePassedBuilds or b.status = "FAILURE")
),

tryjobs_with_status AS (
	SELECT t.*,
	bb.status,
	bb.affected_files,
	bb.affected_files_count
	FROM tryjobs t
	JOIN bb_tryjobs bb USING (id)
),

test_results_base AS (
	SELECT
		tr.result_id,
		tr.name test_name,
		tr.test_id,
		CAST(REGEXP_EXTRACT(exported.id, r'build-(\d+)') as INT64) as build_id,
		IF(tr.test_metadata.location.file_name IS NULL, "", REGEXP_REPLACE(tr.test_metadata.location.file_name, r'^src/', r'//'))  file_name,
		(SELECT v.value FROM tr.variant v where v.key = 'test_suite') AS test_suite,
		(SELECT v.value FROM tr.variant v where v.key = 'builder') AS builder,
		variant_hash,
		expected,
		exonerated,
		status,
		duration,
	FROM chrome-luci-data.chromium.try_test_results tr
	-- Read prev-day and next-day results too to ensure that we have ALL
	-- results of a given CQ attempt.
	WHERE partition_time BETWEEN TIMESTAMP_SUB(@runDate, INTERVAL 1 DAY) and TIMESTAMP_ADD(@runDate, INTERVAL 1 DAY)
		# Exclude third-party tests (except Web Tests) because they test code
		# which isn't in src.git.
		# As of January 2021, this excludes ~2% of test results.
		AND (
			test_metadata.location.file_name NOT LIKE '%/third_party/%'
			OR test_metadata.location.file_name LIKE '//third_party/blink/%'
		)
),

-- Group all test results by patchset, test_id and variant_hash
-- in order to analyze individual test variants in each patchset,
-- and in particular exclude flaky tests.
test_variants_per_ps AS (
	SELECT
		ANY_VALUE(result_id) result_id,
		ANY_VALUE(test_name) test_name,
		ANY_VALUE(test_suite) test_suite,
		ANY_VALUE(builder) builder,
		ANY_VALUE(file_name) file_name,
		ANY_VALUE(affected_files) affected_files,
		ANY_VALUE(affected_files_count) affected_files_count,
		test_id,
		change,
		patchset,
		variant_hash,
		ANY_VALUE(day) day,
		LOGICAL_OR(expected) AND LOGICAL_OR(NOT expected) AS flake,

		# Sometimes ResultDB table misses data. For example, if a test
		# flaked, the table might miss the pass, and it might look like the test
		# has failed. Also sometimes builds are incomplete because they
		# infra-failed or were canceled midway, and the test results do not
		# represent the whole picture. In particular, CANCELED might mean that the
		# "without patch" part didn't finish and test results were not properly
		# exonerated.
		# Thus ensure that the build has failed too.
		LOGICAL_AND(NOT expected) AND LOGICAL_AND(t.status = 'FAILURE') all_unexpected,

		ANY_VALUE(ps_approx_timestamp) AS ps_approx_timestamp,
	FROM tryjobs_with_status t
	JOIN test_results_base tr ON t.id = tr.build_id
	WHERE not exonerated  AND tr.status != 'SKIP' -- not needed for RTS purposes
	GROUP BY change, patchset, test_id, variant_hash
)

SELECT
	rdb.result_id,
	FORMAT("https://crrev.com/c/%d/%d", rdb.change, rdb.patchset) as change_id,
	test_name,
	rdb.variant_hash,
	rdb.test_id,
	rdb.file_name,
	affected_files,
	affected_files_count,
	fr.six_month_fail_count,
	fr.six_month_run_count,
	fr.one_month_fail_count,
	fr.one_month_run_count,
	fr.one_week_fail_count,
	fr.one_week_run_count,
	rdb.all_unexpected as failed,
FROM
	test_variants_per_ps rdb
	LEFT JOIN fail_rate fr
	ON rdb.test_id = fr.test_id AND rdb.variant_hash = fr.variant_hash
WHERE
	(@testSuite = "" OR rdb.test_suite = @testSuite)
	AND (@builder = "" OR rdb.builder = @builder)
	AND fr.six_month_run_count IS NOT NULL
	AND fr.six_month_run_count > 0
	# Remove passes if the option is set
	AND (NOT @onlyTestFailures OR rdb.all_unexpected)
	# Downsample the non-failures if enabled
	AND (@downSample <= 0
		OR rdb.all_unexpected
		OR MOD(FARM_FINGERPRINT(rdb.result_id), @downSample) = 0)
ORDER BY change_id
LIMIT @queryLimit OFFSET @limitOffset
`
