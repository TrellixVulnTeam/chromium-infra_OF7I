// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"

	"go.chromium.org/luci/common/errors"

	"google.golang.org/api/iterator"

	"infra/experimental/loganalysis/gcs"
)

const (
	projectID         = "google.com:stainless-prod"
	bucketID          = "chromeos-test-logs"
	storagePathPrefix = "storage"
)

// LogsInfo summarises all necessary information of a test result's logs.
type LogsInfo struct {
	// LogsURL is the path to the test's logs in the Google Cloud Storage test logs bucket.
	LogsURL string
	// Test is the name of the test that was run.
	Test string
	// FinishedTime is the finished time of the test in the standard time zone of the project.
	FinishedTime string
}

var (
	dateFlag = flag.String("date", "", "test taken date (in the 'yyyymmdd' format)")
	status   = flag.String("status", "FAIL", "status of test results to download (i.e. FAIL/GOOD)")
	board    = flag.String("board", "eve", "reference board name to download test results for")
	limit    = flag.Int("limit", 10, "maximum number of test results to download")

	boardRE = regexp.MustCompile(`[a-z0-9\-]+`)
)

func main() {
	flag.Parse()
	if *dateFlag == "" {
		*dateFlag = time.Now().Format("20060102")
	}
	date, err := time.Parse("20060102", *dateFlag)
	if err != nil {
		log.Fatalf("Cannot parse --date: expected yyyymmdd format, got: %v", err)
	}
	if *status != "FAIL" && *status != "GOOD" {
		log.Fatalf("Cannot parse --status: expected FAIL or GOOD, got: %v", *status)
	}
	if !boardRE.MatchString(*board) {
		log.Fatalf("Cannot parse --board: unexpected characters, got: %v", *board)
	}
	if *limit <= 0 {
		log.Fatalf("Cannot parse --limit: expected a positive integer, got: %v", *limit)
	}
	ctx := context.Background()
	logs, err := FindLogsToDownload(ctx, projectID, date, *status, *board, *limit)
	if err != nil {
		log.Fatalf("Error finding logs to download: %v", err)
	}
	for _, info := range logs {
		contents, err := gcs.ReadFileContents(ctx, bucketID, CreateObjectID(info))
		if err != nil {
			log.Fatalf("Error downloading log contents: %v", err)
		}
		path := filepath.Join(storagePathPrefix, *board, *status, info.FinishedTime+"_"+info.Test+".txt")
		err = SaveContentsToFile(path, contents)
		if err != nil {
			log.Fatalf("Error saving log contents to a file: %v", err)
		}
	}
}

// SaveContentsToFile saves log contents to a local file with the storage path as the combination of attributes for a test.
func SaveContentsToFile(storagePath string, contents []byte) error {
	dir, _ := path.Split(storagePath)
	err := os.MkdirAll(dir, 0770)
	if err != nil {
		return errors.Annotate(err, "create storage directory").Err()
	}
	err = os.WriteFile(storagePath, contents, 0660)
	if err != nil {
		return errors.Annotate(err, "save contents to file").Err()
	}
	return nil
}

// FindLogsToDownload finds logs satisfying specific requirements in the specified project using BigQuery.
func FindLogsToDownload(ctx context.Context, projectID string, date time.Time, status string, board string, limit int) ([]LogsInfo, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, errors.Annotate(err, "create client").Err()
	}
	// Gets logs url and test name (with the required attributes)
	// considering only task tests in the "tast.suite.TestCase" format in the "chromeos-test-logs" project to have the same path to the "log.txt" file.
	q := client.Query(`
			SELECT
				logs_url AS LogsURL,
				test AS Test,
				STRING(finished_time) AS FinishedTime
			FROM ` + "`google.com:stainless-prod.stainless.tests*`" + `
			WHERE
				_TABLE_SUFFIX = @date
				AND status = @status
				AND board = @board
				AND logs_url LIKE '/browse/chromeos-test-logs/%'
				AND test LIKE 'tast.%.%'
			LIMIT
				@limit
	`)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "date", Value: date.Format("20060102")},
		{Name: "status", Value: status},
		{Name: "board", Value: board},
		{Name: "limit", Value: limit},
	}
	logsIterator, err := q.Read(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "read logs information with bigquery").Err()
	}
	var logs []LogsInfo
	for {
		var info LogsInfo
		err := logsIterator.Next(&info)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, errors.Annotate(err, "iterate logs").Err()
		}
		logs = append(logs, info)
	}
	return logs, nil
}

// CreateObjectID creates object ID for a log given its basic information.
// The object ID is used to download log contents.
func CreateObjectID(info LogsInfo) string {
	return strings.TrimPrefix(info.LogsURL, "/browse/"+bucketID+"/") + "/autoserv_test/tast/results/tests/" + info.Test[5:] + "/log.txt"
}
