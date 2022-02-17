// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"hash/fnv"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"

	"cloud.google.com/go/bigquery"

	"go.chromium.org/luci/common/errors"

	"google.golang.org/api/iterator"
)

const (
	projectID = "google.com:stainless-prod"
	bucketID  = "chromeos-test-logs"

	// All downloaded logs from history must be within the bound.
	searchDaysBound = 30
)

// LogsInfo summarises all necessary information of a test result's logs.
type LogsInfo struct {
	// LogsURL is the path to the test's logs in the Google Cloud Storage test logs bucket.
	LogsURL string
	// Board is the board name of the test results that was downloaded.
	Board string
	// Status is the status of test for the downloaded logs (usually FAIL/GOOD).
	Status string
	// Reason is the failure reason of a failing test (or null for a passing test).
	Reason string
}

// logLine records all necessary information of a log line in the target test result for analysis and comparisons.
type logLine struct {
	// original is the original record of a log line.
	original string
	// hash is the hash code of the cleaned log line.
	hash uint64
	// failCount is the count that the log line matches a failing log among all reference failed test results.
	failCount int
	// passCount is the count that the log line matches a passing log among all reference good test results.
	passCount int
}

// referenceURLs records all log URLs of reference test results (i.e. logs from other runs of the same test, used for analysis).
type referenceURLs struct {
	// failTarget is for failing logs with the same board as the target test.
	failTarget []string
	// passTarget is for passing logs with the same board as the target test.
	passTarget []string
	// failOther is for failing logs with different boards from the target test.
	failOther []string
	// passOther is for passing logs with different boards from the target test.
	passOther []string
}

// logSearchOptions lists all the parameters required to find the logs' URLs of all reference tests.
type logSearchOptions struct {
	// date is the beginning searching date, which should be the most recent searching date.
	date time.Time
	// test name required for the reference tests (i.e. test name of the target test).
	test string
	// board of the target test, which should be preferred the same in search.
	board string
	// targetTaskID is the task ID of the target test.
	targetTaskID string
	// reasonFormat is the format of failure reason to look for, expressed as a LIKE expression, e.g. "could not connect to %.%.%.%".
	reasonFormat string
	// requiredNum is the required number of reference tests in either test status.
	requiredNum int
	// searchDaysBound for all reference tests from history.
	searchDaysBound int
}

// analysisOptions lists the parameters inserted by the user and required for analysis.
type analysisOptions struct {
	// test name of the target and reference tests.
	test string
	// logsName of the test for analysis.
	logsName string
	// requiredNum is the required number of reference tests in either test status.
	requiredNum int
}

// analysisResults that will be used to calculate the predictive powers and finally demonstrate to users.
type analysisResults struct {
	// logLines for the target analysis test.
	logLines []logLine
	// totalFails is the total number of failing reference logs in the analysis.
	totalFails int
	// totalPasses is the total number of passing reference logs in the analysis.
	totalPasses int
	// Warnings that should be displayed to the user.
	warnings []string
}

// logAnalysis records the log line analysis statistics.
type logAnalysis struct {
	// hashFileCounts records each log line hash with its corresponding number of occurrence among reference files.
	hashFileCounts map[uint64]int
	// logsAnalyzed is the number of logs that have been analyzed.
	logsAnalyzed int
}

// HighlightedLine is a single line that will be displayed to the user.
type HighlightedLine struct {
	// Content is the original content of a log line.
	Content string
	// Power is the predictive power of a log line.
	Power float64
	// Saturation is calculated by the predictive power to achieve the highlighted colour.
	Saturation int
}

// WebData lists all analysis data that need to be shown in the website for users.
type WebData struct {
	// Warnings are warnings related to smaller number of target reference tests that need to be shown to users.
	Warnings []string
	// HighlightedLines are analysis results of target log lines to print out.
	HighlightedLines []HighlightedLine
}

var (
	dateFlag    = flag.String("date", "", "test taken date (in the 'yyyymmdd' format)")
	test        = flag.String("test", "", "name of the test that was run")
	taskID      = flag.String("id", "", "task id of the test that was run")
	requiredNum = flag.Int("number", 20, "total required number of reference test results in each test status (FAIL/GOOD)")
	port        = flag.String("port", ":3000", "port number of web server (in the ':number' format)")
	logsName    = flag.String("log", "log.txt", "name of the logs file (log.txt/messages in the current version)")

	taskIDRE   = regexp.MustCompile(`[a-z0-9]+`)
	testRE     = regexp.MustCompile(`tast.(\w+).(\w+)(.(\w+))?$`)
	portRE     = regexp.MustCompile(`^:[0-9]+`)
	logsNameRE = regexp.MustCompile(`[a-zA-Z\-_0-9.]+`)

	removal        = regexp.MustCompile("[0-9\t\n]")
	removalHex     = regexp.MustCompile("( )?[+]?0[xX]([0-9a-fA-F]+)(,)?")
	removalBracket = regexp.MustCompile(`(\[(.+)\])|(\{(.+)\})|(\((.+)\))|(\<(.+)\>)`)
	shortenSign    = regexp.MustCompile(`[%]+`)

	tpl *template.Template
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
	if !portRE.MatchString(*port) {
		log.Fatalf("Cannot parse --port: expected a :number format string, got: %v", *port)
	}
	if !logsNameRE.MatchString(*logsName) {
		log.Fatalf("Cannot parse --logsName: expected a string as the log file's name, got: %v", *logsName)
	}

	ctx := context.Background()
	targetLog, err := loadTargetLog(ctx, date, *test, *taskID)
	if err != nil {
		log.Fatalf("Error finding the target log information to download: %v", err)
	}
	reasonFormat := extractReasonFormat(targetLog.Reason)
	logLines, err := readTargetLogLines(ctx, targetLog, *test, *logsName)
	if err != nil {
		log.Fatalf("Error saving log line statistics: %v", err)
	}
	referenceURLs, err := findReferenceLogURLs(ctx, logSearchOptions{date, *test, targetLog.Board, *taskID, reasonFormat, *requiredNum, searchDaysBound})
	if err != nil {
		log.Fatalf("Error finding log urls of all reference logs: %v", err)
	}
	analysisResults := analyzeAllLogs(ctx, logLines, referenceURLs, analysisOptions{*test, *logsName, *requiredNum})
	if analysisResults.totalFails == 0 || analysisResults.totalPasses == 0 {
		log.Fatalf("Error due to zero (failing and/or passing) reference tests: got %v failing and %v passing references", analysisResults.totalFails, analysisResults.totalPasses)
	}

	log.Println("Loading the highlighter web...")
	tpl = template.Must(template.ParseFiles("demo.html"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := WebData{
			Warnings:         analysisResults.warnings,
			HighlightedLines: highlightLines(analysisResults.totalFails, analysisResults.totalPasses, analysisResults.logLines),
		}
		log.Println("Refresh the web page")
		err := tpl.Execute(w, data)
		if err != nil {
			log.Println("Error executing template: ", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})
	log.Fatal(http.ListenAndServe(*port, nil))
}

// extractReasonFormat extracts a LIKE expression matching similar failure reasons with the target test.
func extractReasonFormat(reason string) string {
	reason = strings.ReplaceAll(reason, `\\`, `\\\\`)
	reason = strings.ReplaceAll(reason, `%`, `\\%`)
	reason = strings.ReplaceAll(reason, `_`, `\\_`)
	reason = removal.ReplaceAllString(reason, `%`)
	reason = removalBracket.ReplaceAllString(reason, `%`)
	reason = shortenSign.ReplaceAllString(reason, `%`)
	return reason
}

// highlightLines highlights log lines with different colours and saturations based on their predictive powers.
func highlightLines(totalFails int, totalPasses int, logLines []logLine) []HighlightedLine {
	var res []HighlightedLine
	for _, line := range logLines {
		predPower := calculatePredictivePower(line.failCount, line.passCount, totalFails, totalPasses)
		// Saturation is calculated to make S in HSL colour change steadily from deep red (power=1.0) to lighter red to grey (power=0.5) to lighter green to deep green (power=0) when the predictive power decreases.
		saturation := int(math.Floor(math.Abs(predPower-0.5) * 200))
		// Predictive power is multiplied by 100 and rounded as an integer to show users.
		res = append(res, HighlightedLine{line.original, math.Round(predPower * 100), saturation})
	}
	return res
}

// calculatePredictivePower calculates the predictive power of a log line in the target test.
func calculatePredictivePower(failCount, passCount, totalFails, totalPasses int) float64 {
	return (float64(failCount) / float64(totalFails)) * (1 - (float64(passCount) / float64(totalPasses)))
}

// analyzeAllLogs updates saved statistics of target log lines referring to all reference logs.
func analyzeAllLogs(ctx context.Context, logLines []logLine, urls referenceURLs, analysisOptions analysisOptions) analysisResults {
	var totalFails int
	var totalPasses int
	var warnings []string
	logLines, totalFails, warnings = analyzeLogsForStatus(ctx, "FAIL", logLines, warnings, urls.failTarget, urls.failOther, analysisOptions)
	logLines, totalPasses, warnings = analyzeLogsForStatus(ctx, "GOOD", logLines, warnings, urls.passTarget, urls.passOther, analysisOptions)
	if totalFails < analysisOptions.requiredNum || totalPasses < analysisOptions.requiredNum {
		warnings = append(warnings, "Analysis might be inaccurate, insufficient (passing and/or failing) references overall.")
	}
	return analysisResults{logLines, totalFails, totalPasses, warnings}
}

// analyzeLogsForStatus updates saved statistics of target log lines referring to reference logs from a specific test status.
func analyzeLogsForStatus(ctx context.Context, status string, logLines []logLine, warnings, targetURLs, otherURLs []string, analysisOptions analysisOptions) ([]logLine, int, []string) {
	logAnalysis := newLogAnalysis()
	logAnalysis.transformLogsToHashes(ctx, targetURLs, analysisOptions)
	if logAnalysis.logsAnalyzed < analysisOptions.requiredNum {
		warnings = append(warnings, "Analysis may be inaccurate, insufficient "+status+" references from the target board (required:"+strconv.Itoa(analysisOptions.requiredNum)+", got:"+strconv.Itoa(logAnalysis.logsAnalyzed)+").")
	}
	logAnalysis.transformLogsToHashes(ctx, otherURLs, analysisOptions)
	logLines = compareLogLines(logLines, status == "FAIL", logAnalysis.hashFileCounts)
	return logLines, logAnalysis.logsAnalyzed, warnings
}

// newLogAnalysis creates the initial logAnalysis struct.
func newLogAnalysis() *logAnalysis {
	return &logAnalysis{hashFileCounts: make(map[uint64]int), logsAnalyzed: 0}
}

// transformLogsToHashes transforms all reference test logs to hashes record given a group of log URLs.
func (a *logAnalysis) transformLogsToHashes(ctx context.Context, urls []string, analysisOptions analysisOptions) {
	for _, url := range urls {
		if a.logsAnalyzed >= analysisOptions.requiredNum {
			break
		}
		contents, err := readFileContents(ctx, url, analysisOptions.logsName, analysisOptions.test)
		if err != nil {
			log.Println("Warning: cannot read contents of a test log with logsURL: " + url)
			continue
		}
		addSingleLogToHashes(contents, a.hashFileCounts)
		a.logsAnalyzed += 1
	}
}

// addSingleLogToHashes adds a reference log content to a map with the log line hash as key and its count in all same-status reference logs as value.
func addSingleLogToHashes(contents []byte, hashFileCounts map[uint64]int) {
	hashesRecord := make(map[uint64]bool)
	for _, l := range strings.Split(string(contents), "\n") {
		l = cleanLogLine(l)
		if l == "" {
			continue
		}
		hash := hashLine(l)
		if _, exist := hashesRecord[hash]; !exist {
			hashesRecord[hash] = true
			hashFileCounts[hash] += 1
		}
	}
}

// compareLogLines compares log hashes of the target test with the recorded hashes information of reference tests under a specified test status (FAIL/GOOD).
func compareLogLines(logLines []logLine, testFailed bool, hashFileCounts map[uint64]int) []logLine {
	var res []logLine
	for _, line := range logLines {
		if count, exist := hashFileCounts[line.hash]; exist {
			if testFailed {
				line.failCount += count
			} else {
				line.passCount += count
			}
		}
		res = append(res, line)
	}
	return res
}

// readTargetLogLines saves log line statistics for all lines of the target log with the original contents and the corresponding hash codes.
func readTargetLogLines(ctx context.Context, info LogsInfo, test, logsName string) ([]logLine, error) {
	contents, err := readFileContents(ctx, info.LogsURL, logsName, test)
	if err != nil {
		return nil, errors.Annotate(err, "download log contents").Err()
	}
	var logLines []logLine
	for _, line := range strings.Split(string(contents), "\n") {
		logLines = append(logLines, logLine{line, hashLine(cleanLogLine(line)), 0, 0})
	}
	return logLines, nil
}

// cleanLogLine cleans each log line by removing hexadecimals, numbers, tabs and test time prefixes.
func cleanLogLine(line string) string {
	line = removalHex.ReplaceAllString(line, "")
	line = removal.ReplaceAllString(line, "")
	line = strings.TrimPrefix(line, "--T::.Z [::.] ")
	line = strings.TrimPrefix(line, "--T::.Z ")
	return line
}

// hashLine transforms a string to a hash code.
func hashLine(line string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(line))
	return h.Sum64()
}

// findReferenceLogURLs finds the log URLs of all reference logs.
func findReferenceLogURLs(ctx context.Context, searchOptions logSearchOptions) (referenceURLs, error) {
	var failTargetURLs []string
	var passTargetURLs []string
	var failOtherURLs []string
	var passOtherURLs []string
	remainingFails := searchOptions.requiredNum
	remainingPasses := searchOptions.requiredNum
	currentDate := searchOptions.date
	// Terminates log downloads if the number of files for the specified board reaches expectations or if the test searching date is more than the required bound;
	// otherwise, queries test results with the specific test name from different boards day by day.
	for (remainingFails > 0 || remainingPasses > 0) && currentDate.After(time.Now().AddDate(0, 0, -searchOptions.searchDaysBound)) {
		logs, err := findDailyLogsToDownload(ctx, currentDate, searchOptions.date, searchOptions.test, searchOptions.targetTaskID, searchOptions.reasonFormat, remainingFails > 0, remainingPasses > 0)
		if err != nil {
			return referenceURLs{}, errors.Annotate(err, "find daily reference logs to download").Err()
		}
		currentDate = currentDate.AddDate(0, 0, -1)
		for _, info := range logs {
			if info.Status != "FAIL" && info.Status != "GOOD" {
				continue
			}
			if info.Board == searchOptions.board {
				if info.Status == "FAIL" {
					failTargetURLs = append(failTargetURLs, info.LogsURL)
					remainingFails -= 1
				} else {
					passTargetURLs = append(passTargetURLs, info.LogsURL)
					remainingPasses -= 1
				}
			} else {
				if info.Status == "FAIL" {
					failOtherURLs = append(failOtherURLs, info.LogsURL)
				} else {
					passOtherURLs = append(passOtherURLs, info.LogsURL)
				}
			}
		}
	}
	return referenceURLs{failTargetURLs, passTargetURLs, failOtherURLs, passOtherURLs}, nil
}

// findDailyLogsToDownload finds daily reference logs satisfying specific requirements using BigQuery.
func findDailyLogsToDownload(ctx context.Context, currentDate, targetDate time.Time, test, targetTaskID, reasonFormat string, requireFailTest, requirePassTest bool) ([]LogsInfo, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, errors.Annotate(err, "create client").Err()
	}
	q := client.Query(`
			SELECT
				logs_url AS LogsURL,
				board AS Board,
				status AS Status
			FROM ` + "`google.com:stainless-prod.stainless.tests*`" + `
			WHERE
				_TABLE_SUFFIX = @currentDate
				AND ((@requireFailTest AND status = "FAIL" AND reason LIKE @reasonFormat)
					OR (@requirePassTest AND status = "GOOD"))
				AND logs_url LIKE '/browse/chromeos-test-logs/%'
				AND test = @test
				AND (_TABLE_SUFFIX != @targetDate OR task_id != @targetTaskID)
	`)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "currentDate", Value: currentDate.Format("20060102")},
		{Name: "requireFailTest", Value: requireFailTest},
		{Name: "requirePassTest", Value: requirePassTest},
		{Name: "test", Value: test},
		{Name: "targetDate", Value: targetDate.Format("20060102")},
		{Name: "targetTaskID", Value: targetTaskID},
		{Name: "reasonFormat", Value: reasonFormat},
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

// loadTargetLog loads the target log for analysis.
func loadTargetLog(ctx context.Context, date time.Time, test, taskID string) (LogsInfo, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return LogsInfo{}, errors.Annotate(err, "create client").Err()
	}
	q := client.Query(`
			SELECT
				logs_url AS LogsURL,
				board AS Board,
				status AS Status,
				reason AS Reason
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
		return LogsInfo{}, errors.Annotate(err, "read logs information with bigquery").Err()
	}
	var logs []LogsInfo
	for {
		var info LogsInfo
		err := logsIterator.Next(&info)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return LogsInfo{}, errors.Annotate(err, "iterate logs").Err()
		}
		logs = append(logs, info)
	}
	if len(logs) != 1 {
		return LogsInfo{}, fmt.Errorf("error number of target logs to download: expected a single test result, got: %v", len(logs))
	}
	if logs[0].Status != "FAIL" {
		return LogsInfo{}, fmt.Errorf("error test status: expected a FAIL test, got: %v", logs[0].Status)
	}
	return logs[0], nil
}

// readFileContents reads logs file contents given the basic information of a log.
func readFileContents(ctx context.Context, logsURL, logsName, test string) ([]byte, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "create client").Err()
	}
	r, err := client.Bucket(bucketID).Object(createObjectID(logsURL, test, bucketID, logsName)).NewReader(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "create reader").Err()
	}
	defer r.Close()
	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Annotate(err, "read logs file contents").Err()
	}
	return bytes, nil
}

// createObjectID creates object ID for a log given its basic information.
func createObjectID(logsURL, test, bucketID, logsName string) string {
	return strings.TrimPrefix(logsURL, "/browse/"+bucketID+"/") + "/autoserv_test/tast/results/tests/" + test[5:] + "/" + logsName
}
