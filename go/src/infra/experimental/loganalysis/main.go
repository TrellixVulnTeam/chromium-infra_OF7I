package main

import (
	"context"
	"log"
	"strings"

	"cloud.google.com/go/bigquery"

	"go.chromium.org/luci/common/errors"

	"google.golang.org/api/iterator"

	"infra/experimental/loganalysis/gcs"
)

const (
	projectID = "google.com:stainless-prod"
	bucketID  = "chromeos-test-logs"
)

// LogsInfo summarises all necessary information of a test result's logs.
type LogsInfo struct {
	// LogsURL is the path to the test's logs in the Google Cloud Storage test logs bucket.
	LogsURL string
	// Test is the name of the test that was run.
	Test string
}

func main() {
	ctx := context.Background()
	logs, err := FindLogsToDownload(ctx, projectID)
	if err != nil {
		log.Fatalf("Error finding logs to download: %v", err)
	}
	for _, info := range logs {
		contents, err := gcs.ReadFileContents(ctx, bucketID, CreateObjectID(info))
		if err != nil {
			log.Fatalf("Error downloading log contents: %v", err)
		}
		// TODO(sykq): save log contents to file (rather than printing them out)
		log.Println(string(contents))
	}
}

// FindLogsToDownload finds logs satisfying specific requirements in the specified project using BigQuery.
func FindLogsToDownload(ctx context.Context, projectID string) ([]LogsInfo, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, errors.Annotate(err, "create client").Err()
	}
	// Gets logs url and test name (with limited number for "FAIL" cases for test)
	// considering only task tests in the "tast.suite.TestCase" format in the "chromeos-test-logs" project to have the same path to the "log.txt" file.
	q := client.Query(`
			SELECT
				logs_url AS LogsURL,
				test AS Test
			FROM ` +
		"`google.com:stainless-prod.stainless.tests20211213`" + `
			WHERE
				status = "FAIL"
				AND logs_url LIKE '/browse/chromeos-test-logs/%'
				AND test LIKE 'tast.%.%'
			LIMIT
				50
	`)
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
