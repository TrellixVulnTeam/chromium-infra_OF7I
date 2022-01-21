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
	// FinishedTime is the finished time of the test in the standard time zone of the project.
	FinishedTime string
	// Board is the board name of the test results that was downloaded.
	Board string
	// Status is the status of test for the downloaded logs (usually FAIL/GOOD).
	Status string
}

var (
	dateFlag    = flag.String("date", "", "test taken date (in the 'yyyymmdd' format)")
	board       = flag.String("board", "eve", "reference board name to download test results for")
	limit       = flag.Int("limit", 10, "maximum number of test results to download in each query")
	test        = flag.String("test", "tast.lacros.Basic", "name of the test that was run")
	requiredNum = flag.Int("number", 20, "total required number of test results to download for each test status")

	boardRE = regexp.MustCompile(`[a-z0-9\-]+`)
	testRE  = regexp.MustCompile(`tast.(\w+).(\w+)(.(\w+))?$`)
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
	if !boardRE.MatchString(*board) {
		log.Fatalf("Cannot parse --board: unexpected characters, got: %v", *board)
	}
	if *limit <= 0 {
		log.Fatalf("Cannot parse --limit: expected a positive integer, got: %v", *limit)
	}
	if !testRE.MatchString(*test) {
		log.Fatalf("Cannot parse --test: expected tast.suite.TestCase format, got: %v", *test)
	}
	if *requiredNum <= 0 {
		log.Fatalf("Cannot parse --number: expected a positive integer, got: %v", *requiredNum)
	}

	ctx := context.Background()
	remainingFailNum := *requiredNum
	remainingPassNum := *requiredNum
	currentDate := date
	checkSameBoard := true
	// Terminates log downloads if the number of files reaches expectations; otherwise, queries tests day by day.
	for remainingFailNum > 0 || remainingPassNum > 0 {
		// Checks whether the test searching date is more than 30 days old:
		// if so, for the specified board, we will search the recent results for the same test in other boards;
		// if we have already searched all boards, the log downloads will be terminated.
		if currentDate.Before(time.Now().AddDate(0, 0, -30)) {
			if checkSameBoard {
				checkSameBoard = false
				currentDate = date
			} else {
				break
			}
		}
		logs, err := FindLogsToDownload(ctx, projectID, currentDate, *board, *limit, *test, checkSameBoard, remainingFailNum > 0, remainingPassNum > 0)
		if err != nil {
			log.Fatalf("Error finding logs to download: %v", err)
		}
		currentDate = currentDate.AddDate(0, 0, -1)
		for _, info := range logs {
			if info.Status != "FAIL" && info.Status != "GOOD" {
				continue
			}
			contents, err := gcs.ReadFileContents(ctx, bucketID, CreateObjectID(info, *test))
			if err != nil {
				log.Fatalf("Error downloading log contents: %v", err)
			}
			path := filepath.Join(storagePathPrefix, *test, info.Board, info.Status, info.FinishedTime+".txt")
			err = SaveContentsToFile(path, contents)
			if err != nil {
				log.Fatalf("Error saving log contents to a file: %v", err)
			}
			if info.Status == "FAIL" {
				remainingFailNum -= 1
			} else {
				remainingPassNum -= 1
			}
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
func FindLogsToDownload(ctx context.Context, projectID string, date time.Time, board string, limit int, test string, specifiedBoard bool, requireFailTest bool, requirePassTest bool) ([]LogsInfo, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, errors.Annotate(err, "create client").Err()
	}
	var boardClause string
	if specifiedBoard {
		boardClause = "board = @board"
	} else {
		boardClause = "board != @board"
	}
	q := client.Query(`
			SELECT
				logs_url AS LogsURL,
				STRING(finished_time) AS FinishedTime,
				board AS Board,
				status AS Status
			FROM ` + "`google.com:stainless-prod.stainless.tests*`" + `
			WHERE
				_TABLE_SUFFIX = @date
				AND ((@requireFailTest AND status = "FAIL")
					OR (@requirePassTest AND status = "GOOD"))
				AND (` + boardClause + `)
				AND logs_url LIKE '/browse/chromeos-test-logs/%'
				AND test = @test
			LIMIT
				@limit
	`)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "date", Value: date.Format("20060102")},
		{Name: "requireFailTest", Value: requireFailTest},
		{Name: "requirePassTest", Value: requirePassTest},
		{Name: "board", Value: board},
		{Name: "test", Value: test},
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
func CreateObjectID(info LogsInfo, test string) string {
	return strings.TrimPrefix(info.LogsURL, "/browse/"+bucketID+"/") + "/autoserv_test/tast/results/tests/" + test[5:] + "/log.txt"
}
