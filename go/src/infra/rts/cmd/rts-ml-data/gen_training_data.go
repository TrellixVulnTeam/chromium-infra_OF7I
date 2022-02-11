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
	"io"
	"math"
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

const csv_header = "ResultId,TestName,TestId,FileName,VariantHash,SixMonthRunCount,SixMonthFailCount,OneMonthRunCount,OneMonthFailCount,OneWeekRunCount,OneWeekFailCount,Distance,UseDistance,Failed"

func cmdGenTrainingData(authOpt *auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `gen-training-data -out <path> -from <date> -to <date> -builder <builder>`,
		ShortDesc: "generate features and labels for ML model",
		LongDesc: text.Doc(`
			Benerate features and labels for ML model

			Flags -from -to -out are required.
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
			r.Flags.IntVar(&r.rowCount, "row-count", 100, text.Doc(`
				Max number of rows to process. Default: 100
			`))
			r.Flags.Var(luciflag.Date(&r.startTime), "from", "Fetch results starting from this date; format: yyyy-mm-dd")
			r.Flags.Var(luciflag.Date(&r.endTime), "to", "Fetch results until this date; format: yyyy-mm-dd")
			return r
		},
	}
}

func (r *genTraingData) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := r.ValidateFlags(); err != nil {
		return r.done(err)
	}

	ctx := cli.GetContext(a, r, env)

	entryHashes, err := r.ReadEntryHashes()
	if err != nil && err != io.EOF {
		return r.done(errors.Annotate(err, "failed to read existing result hashes").Err())
	}

	bqClient, err := newBQClient(ctx, auth.NewAuthenticator(ctx, auth.InteractiveLogin, *r.authOpt))
	if err != nil {
		return r.done(errors.Annotate(err, "failed to create BigQuery client").Err())
	}

	rows, err := r.queryResults(ctx, bqClient, entryHashes)
	if err != nil {
		return r.done(errors.Annotate(err, "failed to retrieve query results").Err())
	}

	r.loadInput(ctx)
	r.openCsv()
	defer r.closeCsv()

	fmt.Printf("Calculating distances\n")
	for i, row := range rows {
		if (i+1)%100 == 0 {
			fmt.Printf("Calculating distance on test %d\n", i+1)
		}

		if row.AffectedFilesCount > 100 || row.FileName == "" {
			rows[i].UseDistance = false
		} else {
			// TODO(sshrimp): Group by CL and only calculate this once per CL
			dist := r.calcDistance(row.AffectedFiles, row.FileName)

			if dist < 0 {
				fmt.Printf("WARNING: Test result calculated a negative distance for ResultId = %s\n", row.ResultId)
			}

			valid := !math.IsInf(dist, 0)

			rows[i].Distance = dist
			rows[i].UseDistance = valid
		}
		r.writeCsv(rows[i])
	}

	if err != nil {
		return r.done(errors.Annotate(err, "failed to retrieve query results").Err())
	}

	return 0
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

		fmt.Fprintf(r.file, "%s\n", csv_header)
	} else {
		// TODO(sshrimp): Check the header still matches

		r.file, err = os.OpenFile(r.out, os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *genTraingData) writeCsv(row bqRow) {
	fmt.Fprintf(r.file, "%v,", strings.ReplaceAll(row.ResultId, ",", ";"))
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
	fmt.Fprintf(r.file, "%v", row.Expected)
	fmt.Fprintf(r.file, "\n")
}

func (r *genTraingData) ValidateFlags() error {
	switch {
	case r.out == "":
		return errors.New("-out or -in is required")
	case r.startTime.IsZero():
		return errors.New("-from is required")
	case r.endTime.IsZero():
		return errors.New("-to is required")
	case r.endTime.Before(r.startTime):
		return errors.New("the -to date must not be before the -from date")
	default:
		return nil
	}
}

// Checks for and reads an existing file, returning the first column containing
// result id as a set to be used to avoid duplicating entries
func (r *genTraingData) ReadEntryHashes() (map[string]interface{}, error) {
	hashset := make(map[string]interface{})
	if _, err := os.Stat(r.out); os.IsNotExist(err) {
		return hashset, nil
	}

	fmt.Printf("Reading existing entries\n")
	f, err := os.Open(r.out)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(f)
	line, err := reader.ReadSlice('\n')
	for err == nil {
		hash := string(line[:strings.Index(string(line), ",")])
		hashset[hash] = nil
		line, err = reader.ReadSlice('\n')
	}
	return hashset, err
}

func (r *genTraingData) queryResults(ctx context.Context, bqClient *bigquery.Client, ignoreIds map[string]interface{}) ([]bqRow, error) {
	q := bqClient.Query(filtersQuery)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "rowCount", Value: r.rowCount},
		{Name: "startTime", Value: r.startTime},
		{Name: "endTime", Value: r.endTime},
		{Name: "testSuite", Value: r.testSuite},
		{Name: "builder", Value: r.builder},
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
			return rows, ctx.Err()
		case err != nil:
			return nil, err
		}

		if _, exists := ignoreIds[row.ResultId]; !exists {
			rows = append(rows, *row)

			if err == iterator.Done {
			}
		} else {
			fmt.Printf("Duplicate entry retrieved: %s\n", row.ResultId)
		}
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

func (s *genTraingData) calcDistance(changedFiles []string, testFile string) float64 {
	// This is ineffective and will need to be reworked to run per patchset
	found := false
	distance := 0.0
	s.strategy.RunQuery(changedFiles, func(name string, af rts.Affectedness) (keepGoing bool) {
		if name == testFile {
			found = true
			distance = af.Distance
			return false
		}
		return true
	})
	if found {
		return distance
	} else {
		return math.Inf(1)
	}
}

type genTraingData struct {
	baseCommandRun

	rowCount  int
	startTime time.Time
	endTime   time.Time
	testSuite string
	builder   string

	out      string
	modelDir string
	strategy git.SelectionStrategy
	file     *os.File

	authOpt       *auth.Options
	authenticator *auth.Authenticator
	http          *http.Client
}

type bqRow struct {
	ResultId           string   `bigquery:"result_id"`
	TestName           string   `bigquery:"test_name"`
	TestId             string   `bigquery:"test_id"`
	FileName           string   `bigquery:"file_name"`
	VariantHash        string   `bigquery:"variant_hash"`
	SixMonthFailCount  int64    `bigquery:"six_month_fail_count"`
	SixMonthRunCount   int64    `bigquery:"six_month_run_count"`
	SixMonthFailRate   float32  `bigquery:"six_month_fail_rate"`
	OneMonthFailCount  int64    `bigquery:"one_month_fail_count"`
	OneMonthRunCount   int64    `bigquery:"one_month_run_count"`
	OneMonthFailRate   float32  `bigquery:"one_month_fail_rate"`
	OneWeekFailCount   int64    `bigquery:"one_week_fail_count"`
	OneWeekRunCount    int64    `bigquery:"one_week_run_count"`
	OneWeekFailRate    float32  `bigquery:"one_week_fail_rate"`
	AffectedFilesCount int64    `bigquery:"affected_files_count"`
	AffectedFiles      []string `bigquery:"affected_files"`
	Expected           bool     `bigquery:"failure"`
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
		SUM(IF(day > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY), ARRAY_LENGTH(ds.patchsets_with_failures), 0)) one_month_fail_count,
		SUM(IF(day > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY), ds.run_count, 0)) one_month_run_count,
		SUM(IF(day > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY), ARRAY_LENGTH(ds.patchsets_with_failures), 0)) one_week_fail_count,
		SUM(IF(day > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY), ds.run_count, 0)) one_week_run_count,
	FROM
		chrome-trooper-analytics.test_results.daily_summary ds
	WHERE
		ds.day > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 180 DAY)
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
		WHERE partition_time BETWEEN @startTime AND @endTime
	),

	bb_tryjobs AS (
		SELECT
			id,
			status,
			IF(JSON_EXTRACT(b.output.properties, "$.affected_files.total_count") IS NULL, 0, CAST(CAST(JSON_EXTRACT(b.output.properties, "$.affected_files.total_count") AS FLOAT64) AS INT)) affected_files_count,
			ARRAY(SELECT REGEXP_REPLACE(REPLACE(file, '"', ""), r'^src/', '//') FROM UNNEST(JSON_EXTRACT_ARRAY(b.output.properties, "$.affected_files.first_100")) file) affected_files
		FROM cr-buildbucket.chromium.builds b
		WHERE create_time BETWEEN TIMESTAMP_SUB(@startTime, INTERVAL 1 DAY) AND TIMESTAMP_ADD(@endTime, INTERVAL 1 DAY)
			AND builder.bucket = 'try'
			# Exclude experimental builders because they may fail for reasons
			# unrelated to the CL, and are not required for the CL to land.
			AND STRUCT('cq_experimental', 'true') NOT IN UNNEST(b.tags)
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
		WHERE partition_time BETWEEN TIMESTAMP_SUB(@startTime, INTERVAL 1 DAY) and TIMESTAMP_ADD(@endTime, INTERVAL 1 DAY)
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
	test_name,
	rdb.variant_hash,
	rdb.test_id,
	rdb.file_name,
	affected_files,
	affected_files_count,
	fr.six_month_fail_count,
	fr.six_month_run_count,
	IF(fr.six_month_run_count > 0, fr.six_month_fail_count / fr.six_month_run_count, 0) as six_month_fail_rate,
	fr.one_month_fail_count,
	fr.one_month_run_count,
	IF(fr.one_month_run_count > 0, fr.one_month_fail_count / fr.one_month_run_count, 0) as one_month_fail_rate,
	fr.one_week_fail_count,
	fr.one_week_run_count,
	IF(fr.one_week_run_count > 0, fr.one_week_fail_count / fr.one_week_run_count, 0) as one_week_fail_rate,
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
ORDER BY IF(failed, 1 - (rand() * .001), rand()) DESC
LIMIT @rowCount
`
