// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/maruel/subcommands"
	"google.golang.org/api/iterator"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	luciflag "go.chromium.org/luci/common/flag"

	"go.chromium.org/luci/common/cli"
)

func cmdGenSteFilters(authOpt *auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `gen-ste-filters -out <path> -from <date> -to <date> -builder <builder>`,
		ShortDesc: "generate filter files for stable test exclusion",
		LongDesc: text.Doc(`
			Generate the filter files for stable test exclusion

			Flags -builder -from -to -out are required.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &genSteFilters{}
			r.authOpt = authOpt
			r.Flags.StringVar(&r.builder, "builder", "", "Builder to generate filters for")
			r.Flags.StringVar(&r.out, "out", "", text.Doc(`
				Path to a directory where to write test filter files.
				A file per test target is written, e.g. browser_tests.filter.
			`))
			r.Flags.Var(luciflag.Date(&r.startTime), "from", "Fetch results starting from this date; format: yyyy-mm-dd")
			r.Flags.Var(luciflag.Date(&r.endTime), "to", "Fetch results until this date; format: yyyy-mm-dd")
			r.Flags.Int64Var(&r.minRunCount, "min-run-count", 0, "Minimum times the test must have run to add to the filter")
			return r
		},
	}
}

type genSteFilters struct {
	baseCommandRun

	minRunCount int64
	out         string
	startTime   time.Time
	endTime     time.Time
	builder     string

	authOpt       *auth.Options
	authenticator *auth.Authenticator
	http          *http.Client
}

func (r *genSteFilters) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := r.ValidateFlags(); err != nil {
		return r.done(err)
	}

	ctx := cli.GetContext(a, r, env)

	bqClient, err := newBQClient(ctx, auth.NewAuthenticator(ctx, auth.InteractiveLogin, *r.authOpt))
	if err != nil {
		return r.done(errors.Annotate(err, "failed to create BigQuery client").Err())
	}

	filters, err := r.readQuery(ctx, bqClient)
	if err != nil {
		return r.done(errors.Annotate(err, "failed to retrieve query results").Err())
	}

	err = r.writeFilterFiles(filters)
	if err != nil {
		return r.done(errors.Annotate(err, "failed to create filter files").Err())
	}

	return 0
}

func (r *genSteFilters) ValidateFlags() error {
	switch {
	case r.builder == "":
		return errors.New("-builder is required")
	case r.out == "":
		return errors.New("-out is required")
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

func (r *genSteFilters) readQuery(ctx context.Context, bqClient *bigquery.Client) (map[string][]string, error) {
	filterFiles := map[string][]string{}

	q := bqClient.Query(filtersQuery)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "startTime", Value: r.startTime},
		{Name: "endTime", Value: r.endTime},
		{Name: "builder", Value: r.builder},
	}

	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}

	row := &bqFilterRow{}
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		// Read the next row.
		switch err := it.Next(row); {
		case err == iterator.Done:
			return filterFiles, ctx.Err()
		case err != nil:
			return nil, err
		}

		if row.FailCount == 0 && row.RunCount >= r.minRunCount {
			filterFiles[row.TestSuite] = append(filterFiles[row.TestSuite], row.TestName)
		}
	}
}

func (r *genSteFilters) writeFilterFiles(filterFiles map[string][]string) error {
	for suite, tests := range filterFiles {
		// The variant if flattened into key:value strings, need to remove the prefix
		filename := strings.TrimPrefix(suite, "test_suite:") + ".filter"
		folder := r.out + "/" + r.builder

		exists, err := folderExists(folder)
		if err != nil {
			return err
		}
		if !exists {
			os.MkdirAll(folder, 0777)
		}

		f, err := os.Create(r.out + "/" + r.builder + "/" + filename)
		if err != nil {
			return err
		}

		for _, test_name := range tests {
			f.WriteString("-" + test_name + "\n")
		}
	}
	return nil
}

func folderExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

type bqFilterRow struct {
	TestName  string `bigquery:"test_name"`
	TestId    string `bigquery:"test_id"`
	FailCount int64  `bigquery:"fail_count"`
	RunCount  int64  `bigquery:"run_count"`
	TestSuite string `bigquery:"test_suite"`
}

const filtersQuery = `
WITH ids_to_names AS (
    SELECT
        tr.test_id,
        ANY_VALUE((SELECT ANY_VALUE(t.value) FROM UNNEST(tr.tags) t WHERE t.key = 'test_name')) as test_name
    FROM chrome-luci-data.chromium.try_test_results tr
    WHERE tr.partition_time BETWEEN TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY) AND CURRENT_TIMESTAMP()
    GROUP BY tr.test_id
)

SELECT
    ANY_VALUE(n.test_name) test_name,
    s.test_id AS test_id,
    SUM(ARRAY_LENGTH(s.patchsets_with_failures)) fail_count,
    SUM(run_count) run_count,
    (SELECT * FROM s.testVariant.variant v WHERE v LIKE 'test_suite:%') test_suite
FROM chrome-trooper-analytics.test_results.daily_summary s, ids_to_names n
WHERE
    s.day BETWEEN @startTime AND @endTime
    AND s.test_id = n.test_id
    AND ('builder:' || @builder) in UNNEST(s.testVariant.variant)
GROUP BY test_id, test_suite
HAVING fail_count = 0`
