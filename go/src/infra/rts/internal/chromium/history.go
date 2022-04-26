// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package chromium

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/errors"
	luciflag "go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/sync/parallel"
)

// The range of the number of changed files to enable RTS.
// If the number of changed files is not in this range, then RTS is disabled.
// Note that 99.3% of developer-authored git commits change <= 100 files, see bit.ly/chromium-rts
const (
	MinChangedFiles = 1
	MaxChangedFiles = 100
)

type baseHistoryRun struct {
	out          string
	startTime    time.Time
	endTime      time.Time
	builderRegex string
	testIDRegex  string
	clOwner      string

	authOpt       *auth.Options
	authenticator *auth.Authenticator
	http          *http.Client
}

func NewBQClient(ctx context.Context, auth *auth.Authenticator) (*bigquery.Client, error) {
	http, err := auth.Client()
	if err != nil {
		return nil, err
	}
	return bigquery.NewClient(ctx, "chrome-rts", option.WithHTTPClient(http))
}

func (r *baseHistoryRun) RegisterBaseFlags(fs *flag.FlagSet) {
	fs.StringVar(&r.out, "out", "", "Path to the output directory")
	fs.Var(luciflag.Date(&r.startTime), "from", "Fetch results starting from this date; format: 2020-01-15")
	fs.Var(luciflag.Date(&r.endTime), "to", "Fetch results until this date; format: 2020-01-15")
	fs.StringVar(&r.builderRegex, "builder", ".*", "A regular expression for builder. Implicitly wrapped with ^ and $.")
	fs.StringVar(&r.testIDRegex, "test", ".*", "A regular expression for test. Implicitly wrapped with ^ and $.")
	fs.StringVar(&r.clOwner, "cl-owner", "", "CL owner, e.g. someone@chromium.org")
}

func (r *baseHistoryRun) ValidateBaseFlags() error {
	switch {
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

// Init initializes r. Validates flags as well.
func (r *baseHistoryRun) Init(ctx context.Context) error {
	if err := r.ValidateBaseFlags(); err != nil {
		return err
	}

	r.authenticator = auth.NewAuthenticator(ctx, auth.InteractiveLogin, *r.authOpt)
	var err error
	r.http, err = r.authenticator.Client()
	return err
}

// runAndFetchResults runs the BigQuery query and saves results to r.out directory,
// as GZIP-compressed JSON Lines files.
func (r *baseHistoryRun) runAndFetchResults(ctx context.Context, sql string, extraParams ...bigquery.QueryParameter) error {
	logging.Infof(ctx, "starting a BigQuery query...\n")
	table, err := r.runQuery(ctx, sql, extraParams...)
	if err != nil {
		return err
	}

	logging.Infof(ctx, "fetching results...\n")
	return r.fetchResults(ctx, table)
}

// runQuery runs the query and returns the table with results.
func (r *baseHistoryRun) runQuery(ctx context.Context, sql string, extraParams ...bigquery.QueryParameter) (*bigquery.Table, error) {
	client, err := NewBQClient(ctx, r.authenticator)
	if err != nil {
		return nil, err
	}

	// Prepare the query.

	prepRe := func(rgx string) string {
		if rgx == "" || rgx == ".*" {
			return ""
		}
		return fmt.Sprintf("^(%s)$", rgx)
	}

	q := client.Query(sql)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "startTime", Value: r.startTime},
		{Name: "endTime", Value: r.endTime},
		{Name: "builderRegexp", Value: prepRe(r.builderRegex)},
		{Name: "testIdRegexp", Value: prepRe(r.testIDRegex)},
		{Name: "minChangedFiles", Value: MinChangedFiles},
		{Name: "maxChangedFiles", Value: MaxChangedFiles},
		{Name: "clOwner", Value: r.clOwner},
	}
	q.Parameters = append(q.Parameters, extraParams...)

	// Run the query.

	job, err := q.Run(ctx)
	if err != nil {
		return nil, err
	}
	logging.Infof(ctx, "BigQuery job: https://console.cloud.google.com/bigquery?project=chrome-rts&page=queryresults&j=bq:US:%s", job.ID())
	if err := waitForSuccess(ctx, job); err != nil {
		return nil, err
	}

	cfg, err := job.Config()
	if err != nil {
		return nil, err
	}
	return cfg.(*bigquery.QueryConfig).Dst, nil
}

// fetchResults fetches the table to GZIP-compressed JSON Lines files in r.out
// directory.
func (r *baseHistoryRun) fetchResults(ctx context.Context, table *bigquery.Table) error {
	// The fetching processing is done in two phases:
	// 1. Extract the table to GCS.
	//    This also takes care of the converting from tabular format to a file format.
	// 2. Download the prepared files to the destination directory.

	if err := PrepareOutDir(r.out, "*.jsonl.gz"); err != nil {
		return err
	}

	// Extract the table to GCS.
	bucketName := "chrome-rts"
	dirName := fmt.Sprintf("tmp/extract-%s", table.TableID)
	gcsDir := fmt.Sprintf("gs://%s/%s/", bucketName, dirName)
	logging.Infof(ctx, "extracting results to %s", gcsDir)
	gcsRef := &bigquery.GCSReference{
		// Start the object name with a date, so that the user can merge
		// data directories if needed.
		URIs:              []string{fmt.Sprintf("%s%s-*.jsonl.gz", gcsDir, r.startTime.Format("2006-01-02"))},
		DestinationFormat: bigquery.JSON,
		Compression:       bigquery.Gzip,
	}
	job, err := table.ExtractorTo(gcsRef).Run(ctx)
	if err != nil {
		return err
	}
	if err := waitForSuccess(ctx, job); err != nil {
		return errors.Annotate(err, "extract job %q failed", job.ID()).Err()
	}

	// Fetch the extracted files from GCS to the file system.
	gcs, err := storage.NewClient(ctx, option.WithHTTPClient(r.http))
	if err != nil {
		return err
	}
	bucket := gcs.Bucket(bucketName)
	// Find all files in the temp GCS dir and fetch them all, concurrently.
	iter := bucket.Objects(ctx, &storage.Query{Prefix: dirName})
	return parallel.WorkPool(100, func(work chan<- func() error) {
		for {
			objAttrs, err := iter.Next()
			switch {
			case err == iterator.Done:
				return
			case err != nil:
				work <- func() error { return err }
				return
			}

			// Fetch the file.
			work <- func() error {
				// Prepare the source.
				rd, err := bucket.Object(objAttrs.Name).NewReader(ctx)
				if err != nil {
					return err
				}
				defer rd.Close()

				// Prepare the sink.
				f, err := os.Create(filepath.Join(r.out, path.Base(objAttrs.Name)))
				if err != nil {
					return err
				}
				defer f.Close()

				_, err = io.Copy(f, rd)
				return err
			}
		}
	})
}

func waitForSuccess(ctx context.Context, job *bigquery.Job) error {
	st, err := job.Wait(ctx)
	if err != nil {
		return err
	}
	return st.Err()
}
