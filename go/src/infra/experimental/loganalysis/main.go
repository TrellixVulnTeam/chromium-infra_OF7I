// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"hash/fnv"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"

	"go.chromium.org/luci/common/errors"

	"google.golang.org/api/iterator"

	"infra/experimental/loganalysis/collection/gcs"
)

const (
	projectID = "google.com:stainless-prod"

	// All log lines with predictive power higher than the set threshold are highlighted in red.
	predPowerThreshold = 0.98
)

// LogsInfo summarises all necessary information of a test result's logs.
type LogsInfo struct {
	// LogsURL is the path to the test's logs in the Google Cloud Storage test logs bucket.
	LogsURL string
	// Board is the board name of the test results that was downloaded.
	Board string
	// Status is the status of test for the downloaded logs (usually FAIL/GOOD).
	Status string
}

// LogLine records all necessary information of a log line in the target test result for analysis and comparisons.
type LogLine struct {
	// Original is the original record of a log line.
	Original string
	// Hash is the hash code of the cleaned log line.
	Hash uint64
	// FailCount is the count that the log line matches a failing log among all reference failed test results.
	FailCount int
	// PassCount is the count that the log line matches a passing log among all reference good test results.
	PassCount int
}

var (
	dateFlag    = flag.String("date", "", "test taken date (in the 'yyyymmdd' format)")
	test        = flag.String("test", "", "name of the test that was run")
	taskID      = flag.String("id", "", "task id of the test that was run")
	requiredNum = flag.Int("number", 20, "total required number of reference test results in each test status (FAIL/GOOD)")

	taskIDRE = regexp.MustCompile(`[a-z0-9]+`)
	testRE   = regexp.MustCompile(`tast.(\w+).(\w+)(.(\w+))?$`)

	removalHex = regexp.MustCompile("( )?[+]?0[xX]([0-9a-fA-F]+)(,)?")
	removal    = regexp.MustCompile("[0-9\t\n]")

	statuses = []string{"GOOD", "FAIL"}
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
	if *test == "" {
		log.Fatalf("Cannot parse --test: please insert the target test name, got: %v", *test)
	}
	if !testRE.MatchString(*test) {
		log.Fatalf("Cannot parse --test: expected tast.suite.TestCase format, got: %v", *test)
	}
	if *taskID == "" {
		log.Fatalf("Cannot parse --taskID: please insert the target task ID, got: %v", *taskID)
	}
	if !taskIDRE.MatchString(*taskID) {
		log.Fatalf("Cannot parse --taskID: expected a string combined with lower-case characters and numbers, got: %v", *taskID)
	}
	if *requiredNum <= 0 {
		log.Fatalf("Cannot parse --number: expected a positive integer, got: %v", *requiredNum)
	}

	ctx := context.Background()
	targetLog, err := downloadTargetLog(ctx, date, *test, *taskID)
	if err != nil {
		log.Fatalf("Error finding the target log information to download: %v", err)
	}
	logLines, err := saveLineStatistics(ctx, targetLog, *test, gcs.LogsName)
	if err != nil {
		log.Fatalf("Error saving log line statistics: %v", err)
	}

	var totalFails int
	var totalPasses int
	for _, status := range statuses {
		path := filepath.Join(gcs.StoragePathPrefix, *test, status)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Fatalf("Error in finding any reference %v test results from all boards for analysis: %v", status, err)
		}
		var totalCount int
		logLines, totalCount, err = analyzeTargetBoardLogs(path, targetLog.Board, status, logLines)
		if err != nil {
			log.Fatalf("Error in analyzing reference logs from the target board: %v", err)
		}
		if totalCount < *requiredNum {
			logLines, totalCount, err = analyzeOtherBoardsLogs(path, targetLog.Board, status, logLines, totalCount, *requiredNum)
			if err != nil {
				log.Fatalf("Error in analyzing reference logs from other boards: %v", err)
			}
		}
		if totalCount < *requiredNum {
			log.Println("Warning: not enough number of reference " + status + " test results overall, expected:" + strconv.Itoa(*requiredNum) + ", got:" + strconv.Itoa(totalCount))
		}
		if status == "FAIL" {
			totalFails = totalCount
		} else {
			totalPasses = totalCount
		}
	}
	highlightImportantLines(totalFails, totalPasses, logLines, predPowerThreshold)
}

// highlightImportantLines prints and highlights log lines with higher predictive power than the set threshold.
func highlightImportantLines(totalFails, totalPasses int, logLines []LogLine, threshold float32) {
	fmt.Println("Please delete the local storage of the downloaded logs in this analysis for result accuracy in the future!")
	fmt.Println("Highlighted Results:")
	for _, line := range logLines {
		if calculatePredictivePower(line.FailCount, line.PassCount, totalFails, totalPasses) > threshold {
			fmt.Println(string("\033[31m"), line.Original)
		} else {
			fmt.Println(string("\033[0m"), line.Original)
		}
	}
}

// analyzeOtherBoardsLogs updates saved statistics of target log lines referring to test logs from other boards when the number of stored test results from the target board is not enough.
func analyzeOtherBoardsLogs(destDir, board, status string, logLines []LogLine, totalCount, requiredNum int) ([]LogLine, int, error) {
	log.Println("Warning: not enough number of reference " + status + " test results from the target board, expected:" + strconv.Itoa(requiredNum) + ", got:" + strconv.Itoa(totalCount))
	boardsDir, err := ioutil.ReadDir(destDir + "/")
	if err != nil {
		return nil, 0, errors.Annotate(err, "read storage directory with the specified test name").Err()
	}
	boardNames := listOtherBoardNames(boardsDir, board)
	var count int
	for _, boardName := range boardNames {
		logLines, count, err = updateLineStatistics(filepath.Join(destDir, boardName), status, logLines)
		log.Println("Warning: get " + strconv.Itoa(count) + " reference " + status + " test results from board " + boardName + " for the analysis")
		totalCount += count
	}
	return logLines, totalCount, nil
}

// analyzeTargetBoardLogs updates saved statistics of target log lines referring to test logs from the target board.
func analyzeTargetBoardLogs(path, board, status string, logLines []LogLine) ([]LogLine, int, error) {
	if _, err := os.Stat(filepath.Join(path, board)); os.IsNotExist(err) {
		log.Println("Warning: 0 reference " + status + " test result from the target board " + board + " for the analysis")
		return logLines, 0, nil
	}
	logLines, count, err := updateLineStatistics(filepath.Join(path, board), status, logLines)
	if err != nil {
		return nil, 0, errors.Annotate(err, "update log lines to count matching numbers").Err()
	}
	return logLines, count, nil
}

// calculatePredictivePower calculates the predictive power of a log line in the target test.
func calculatePredictivePower(failCount, passCount, totalFails, totalPasses int) float32 {
	return (float32(failCount) / float32(totalFails)) * (1 - (float32(passCount) / float32(totalPasses)))
}

// saveLineStatistics saves log line statistics for all lines of the target log with the original contents and the corresponding hash codes.
func saveLineStatistics(ctx context.Context, info LogsInfo, test, logsName string) ([]LogLine, error) {
	contents, err := gcs.ReadFileContents(ctx, gcs.BucketID, gcs.CreateObjectID(info.LogsURL, test, gcs.BucketID, logsName))
	if err != nil {
		return nil, errors.Annotate(err, "download log contents").Err()
	}
	var logLines []LogLine
	for _, line := range strings.Split(string(contents), "\n") {
		logLines = append(logLines, LogLine{line, hashLine(cleanLogLine(line)), 0, 0})
	}
	return logLines, nil
}

// updateLineStatistics updates the target log lines' related statistics by counting the matching numbers for each line in the target test result compared with reference files in the given path.
func updateLineStatistics(path, status string, logLines []LogLine) ([]LogLine, int, error) {
	filesInfo, err := ioutil.ReadDir(path + "/")
	if err != nil {
		return nil, 0, errors.Annotate(err, "read reference test results' directory").Err()
	}
	fileCountByHash, err := transformFilesToHashes(path, filesInfo)
	if err != nil {
		return nil, 0, errors.Annotate(err, "transform reference files to a map of hashes").Err()
	}
	logLines = compareLogLines(logLines, status == "FAIL", fileCountByHash)
	return logLines, len(filesInfo), nil
}

// compareLogLines compares log hashes of the target test with the map recorded hashes information of reference tests under a specified test status (FAIL/GOOD).
func compareLogLines(logLines []LogLine, testFailed bool, fileCountByHash map[uint64]int) []LogLine {
	var res []LogLine
	for _, line := range logLines {
		if count, exist := fileCountByHash[line.Hash]; exist {
			if testFailed {
				line.FailCount += count
			} else {
				line.PassCount += count
			}
		}
		res = append(res, line)
	}
	return res
}

// transformFilesToHashes transforms all reference test logs to a map with the log line hash as key and the corresponding document indexes list as value.
func transformFilesToHashes(path string, filesInfo []fs.FileInfo) (map[uint64]int, error) {
	res := make(map[uint64]int)
	for _, fileInfo := range filesInfo {
		hashesRecord := make(map[uint64]bool)
		file, err := os.Open(filepath.Join(path, fileInfo.Name()))
		if err != nil {
			return nil, errors.Annotate(err, "open file").Err()
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			l := cleanLogLine(scanner.Text())
			if l == "" {
				continue
			}
			hash := hashLine(l)
			if _, exist := hashesRecord[hash]; !exist {
				hashesRecord[hash] = true
				if _, exist := res[hash]; exist {
					res[hash] += 1
				} else {
					res[hash] = 1
				}
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, errors.Annotate(err, "scan file").Err()
		}
	}
	return res, nil
}

// cleanLogLine cleans each log line by removing hexadecimals, numbers, tabs and test time prefixes.
func cleanLogLine(line string) string {
	l := removalHex.ReplaceAllString(line, "")
	l = removal.ReplaceAllString(l, "")
	l = strings.TrimPrefix(l, "--T::.Z [::.] ")
	l = strings.TrimPrefix(l, "--T::.Z ")
	return l
}

// hashLine transforms a string to a hash code.
func hashLine(line string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(line))
	return h.Sum64()
}

// listOtherBoardNames lists board names that saved in the test results' storage while different to the target board.
func listOtherBoardNames(boards []fs.FileInfo, targetBoardName string) []string {
	var boardNames []string
	for _, b := range boards {
		if b.Name() != targetBoardName {
			boardNames = append(boardNames, b.Name())
		}
	}
	return boardNames
}

// downloadTargetLog downloads the target log for analysis satisfying specific requirements in the specified project.
func downloadTargetLog(ctx context.Context, date time.Time, test, taskID string) (LogsInfo, error) {
	nilInfo := LogsInfo{}
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nilInfo, errors.Annotate(err, "create client").Err()
	}
	q := client.Query(`
			SELECT
				logs_url AS LogsURL,
				board AS Board,
				status AS Status
			FROM ` + "`google.com:stainless-prod.stainless.tests*`" + `
			WHERE
				_TABLE_SUFFIX = @date
				AND test = @test
				AND task_id = @taskID
	`)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "date", Value: date.Format("20060102")},
		{Name: "test", Value: test},
		{Name: "taskID", Value: taskID},
	}
	logsIterator, err := q.Read(ctx)
	if err != nil {
		return nilInfo, errors.Annotate(err, "read logs information with bigquery").Err()
	}
	var logs []LogsInfo
	for {
		var info LogsInfo
		err := logsIterator.Next(&info)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nilInfo, errors.Annotate(err, "iterate logs").Err()
		}
		logs = append(logs, info)
	}
	if len(logs) != 1 {
		return nilInfo, fmt.Errorf("error number of target logs to download: expected a single test result, got: %v", len(logs))
	}
	if logs[0].Status != "FAIL" {
		return nilInfo, fmt.Errorf("error test status: expected a FAIL test, got: %v", logs[0].Status)
	}
	return logs[0], nil
}
