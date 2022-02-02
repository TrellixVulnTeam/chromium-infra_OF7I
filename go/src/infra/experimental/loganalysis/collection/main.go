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
	"time"

	"cloud.google.com/go/bigquery"

	"go.chromium.org/luci/common/errors"

	"google.golang.org/api/iterator"

	"infra/experimental/loganalysis/collection/gcs"
)

const (
	projectID = "google.com:stainless-prod"

	// All downloaded logs from history must be within the bound.
	searchDaysBound = 30
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
	remainingFails := *requiredNum
	remainingPasses := *requiredNum
	currentDate := date
	var otherBoardsFailInfo []LogsInfo
	var otherBoardsPassInfo []LogsInfo
	// Terminates log downloads if the number of files for the specified board reaches expectations or if the test searching date is more than the required bound;
	// otherwise, queries test results with the specific test name from different boards day by day.
	for (remainingFails > 0 || remainingPasses > 0) && currentDate.After(time.Now().AddDate(0, 0, -searchDaysBound)) {
		logs, err := findLogsToDownload(ctx, currentDate, *limit, *test, remainingFails > 0, remainingPasses > 0)
		if err != nil {
			log.Fatalf("Error finding logs to download: %v", err)
		}
		currentDate = currentDate.AddDate(0, 0, -1)
		for _, info := range logs {
			if info.Status != "FAIL" && info.Status != "GOOD" {
				continue
			}
			if info.Board == *board {
				success, err := downloadLogContents(ctx, info, *test, gcs.LogsName, gcs.StoragePathPrefix)
				if err != nil {
					log.Fatalf("Error downloading log contents of the target board locally: %v", err)
				}
				if success {
					if info.Status == "FAIL" {
						remainingFails -= 1
					} else {
						remainingPasses -= 1
					}
				}
			} else {
				if info.Status == "FAIL" {
					otherBoardsFailInfo = append(otherBoardsFailInfo, info)
				} else {
					otherBoardsPassInfo = append(otherBoardsPassInfo, info)
				}
			}
		}
	}
	// If the required number of logs under a specific status is not satisfied, the most recent test logs from other boards will be downloaded to satisfy the number requirement.
	if remainingFails > 0 {
		err = downloadOtherBoardsLogs(ctx, otherBoardsFailInfo, remainingFails, *test, gcs.LogsName, gcs.StoragePathPrefix)
		if err != nil {
			log.Fatalf("Error saving failing logs of a different board: %v", err)
		}
	}
	if remainingPasses > 0 {
		err = downloadOtherBoardsLogs(ctx, otherBoardsPassInfo, remainingPasses, *test, gcs.LogsName, gcs.StoragePathPrefix)
		if err != nil {
			log.Fatalf("Error saving passing logs of a different board: %v", err)
		}
	}
}

// downloadOtherBoardsLogs downloads logs contents of other boards locally if the downloaded number from the target board is not enough.
func downloadOtherBoardsLogs(ctx context.Context, otherBoardsInfo []LogsInfo, requiredNum int, test, logsName, destDir string) error {
	for _, info := range otherBoardsInfo {
		if requiredNum <= 0 {
			break
		}
		success, err := downloadLogContents(ctx, info, test, logsName, destDir)
		if err != nil {
			return errors.Annotate(err, "download log contents of a different board").Err()
		}
		if success {
			requiredNum -= 1
		}
	}
	return nil
}

// downloadLogContents downloads specific logs content locally given the information of a test log.
func downloadLogContents(ctx context.Context, info LogsInfo, test, logsName, destDir string) (bool, error) {
	contents, err := gcs.ReadFileContents(ctx, gcs.BucketID, gcs.CreateObjectID(info.LogsURL, test, gcs.BucketID, logsName))
	if err != nil {
		log.Println("Warning: cannot read contents of a test log with logsURL " + info.LogsURL)
		return false, nil
	}
	err = storeContentsToFile(filepath.Join(gcs.StoragePathPrefix, test, info.Status, info.Board, info.FinishedTime+".txt"), contents)
	if err != nil {
		return false, errors.Annotate(err, "store contents to a file").Err()
	}
	return true, nil
}

// storeContentsToFile stores contents to a local file with the storage path as the combination of attributes for a test.
func storeContentsToFile(destDir string, contents []byte) error {
	dir, _ := path.Split(destDir)
	err := os.MkdirAll(dir, 0770)
	if err != nil {
		return errors.Annotate(err, "create storage directory").Err()
	}
	err = os.WriteFile(destDir, contents, 0660)
	if err != nil {
		return errors.Annotate(err, "write contents to file").Err()
	}
	return nil
}

// findLogsToDownload finds logs satisfying specific requirements in the specified project using BigQuery.
func findLogsToDownload(ctx context.Context, date time.Time, limit int, test string, requireFailTest, requirePassTest bool) ([]LogsInfo, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, errors.Annotate(err, "create client").Err()
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
				AND logs_url LIKE '/browse/chromeos-test-logs/%'
				AND test = @test
			LIMIT
				@limit
	`)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "date", Value: date.Format("20060102")},
		{Name: "requireFailTest", Value: requireFailTest},
		{Name: "requirePassTest", Value: requirePassTest},
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
