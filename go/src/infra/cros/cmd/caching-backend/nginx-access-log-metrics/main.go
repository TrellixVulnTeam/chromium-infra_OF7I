// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"

	"infra/cros/cmd/caching-backend/nginx-access-log-metrics/internal/bquploader"
	"infra/cros/cmd/caching-backend/nginx-access-log-metrics/internal/filetailer"
)

type record struct {
	timestamp     time.Time
	hostname      string
	clientIP      string
	httpMethod    string
	path          string
	status        int
	bodyBytesSent int
	expectedSize  int
	requestTime   float64
	cacheStatus   string
}

var (
	svcAcctJSONPath = flag.String("service-account-json", "", "Path to JSON file with service account credentials to use")
	projectID       = flag.String("project-id", "cros-lab-servers", "ID of the cloud projecdt to upload metrics data to")
	dataset         = flag.String("dataset", "caching_backend", "Dataset name of the BigQuery tables")
	tableName       = flag.String("table", "access_log", "BigQuery table name")
	inputLogFile    = flag.String("input-log-file", "/var/log/nginx/gs-cache.access.log", "Nginx access log for gs_cache")
)

func main() {
	if err := innerMain(); err != nil {
		log.Fatalf("Exiting due to an error: %s", err)
	}
	log.Printf("Exiting successfully")
}

func innerMain() error {
	flag.Parse()

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	ctx := cancelOnSignals(context.Background())

	t := bquploader.TargetTable{
		ProjectID: *projectID,
		Dataset:   *dataset,
		TableName: *tableName,
	}
	uploader, err := bquploader.NewUploader(t, 10*time.Minute, option.WithCredentialsFile(*svcAcctJSONPath))
	if err != nil {
		return err
	}
	defer uploader.Close()

	tailer, err := filetailer.New(*inputLogFile)
	if err != nil {
		return err
	}
	defer tailer.Close()

	go func() {
		for tailer.Scan() {
			if r := parseLine(tailer.Text()); r != nil {
				r.hostname = hostname
				uploader.QueueRecord(r)
			}
		}
	}()
	<-ctx.Done()
	return nil
}

// See https://chromium.googlesource.com/infra/infra/+/refs/heads/main/go/src/infra/cros/cmd/caching-backend/conf-creator/conf_templates.go#55
// for the detailed log format definition.
// An example log line:
// 127.0.0.1 - - [2021-06-09T20:24:39+00:00] "GET /download/abc HTTP/1.1" 200 369 "-" 0.000 "-" "curl/7.66.0" "-" -
const logLinePattern = `^(?P<client_ip>\S+) - \S+ \[(?P<timestamp>[^\]]+)\] ` +
	`"(?P<http_method>\S+) (?P<path>\S+)[^"]*" (?P<status>\d+) (?P<body_bytes_sent>\d+) "(?P<upstream_http_content_length>\S+)" ` +
	`(?P<request_time>\d+[\.\d]+) "[^"]+" "[^"]+" "[^"]+" (?P<cache_status>\S+)`

var logRegex = regexp.MustCompile(logLinePattern)

// parseLine parses a log line and generate a record if the line matches the
// predefined pattern.
func parseLine(line string) *record {
	matches := logRegex.FindStringSubmatch(line)
	if len(matches) == 0 {
		return nil
	}

	// groups is a map from group name to its value.
	groups := make(map[string]string)
	for i, group := range logRegex.SubexpNames() {
		// Handle the case of "entire matching group" (i.e. index == 0) and
		// unamed group.
		if i == 0 || group == "" {
			continue
		}
		groups[group] = matches[i]
	}
	status, err := strconv.Atoi(groups["status"])
	if err != nil {
		log.Printf("Parse status: %s", err)
		return nil
	}
	// Ignore client side errors (4xx) except NotFound(404).
	// We may receive many client bad requests from some service scanner, so
	// ignore them.
	if status >= 400 && status < 500 && status != 404 {
		log.Printf("Ignore client error %d", status)
		return nil
	}

	// Ignore non cache related requests.
	path := groups["path"]
	if ignoredPath(path) {
		log.Printf("Ignore path %q", path)
		return nil
	}
	timestamp, err := time.Parse("2006-01-02T15:04:05-07:00", groups["timestamp"])
	if err != nil {
		log.Printf("Parse timestamp: %s", err)
		return nil
	}
	requestTime, err := strconv.ParseFloat(groups["request_time"], 64)
	if err != nil {
		log.Printf("Parse request time: %s", err)
		return nil
	}
	bodyBytesSent, err := strconv.Atoi(groups["body_bytes_sent"])
	if err != nil {
		log.Printf("Parse body_bytes_sent: %s", err)
		return nil
	}

	var expectedSize int
	if groups["upstream_http_content_length"] == "-" {
		// We use -1 to indicate that the size is unknown.
		expectedSize = -1
	} else {
		expectedSize, err = strconv.Atoi(groups["upstream_http_content_length"])
		if err != nil {
			log.Printf("Parse upstream_http_content_length: %s", err)
			return nil
		}
	}

	return &record{
		timestamp:     timestamp,
		clientIP:      groups["client_ip"],
		httpMethod:    groups["http_method"],
		status:        status,
		path:          path,
		bodyBytesSent: bodyBytesSent,
		expectedSize:  expectedSize,
		requestTime:   requestTime,
		cacheStatus:   groups["cache_status"],
	}
}

// ignoredPath ignores path/URL not related to caching.
func ignoredPath(path string) bool {
	switch {
	case path == "/":
		return true
	case path == "/static/quick-provision":
		return true
	case path == "/download/chromeos-image-archive":
		return true
	case path == "/download/chromeos-releases":
		return true
	case strings.HasPrefix(path, "/check_health"):
		return true
	case strings.HasPrefix(path, "/stage"):
		return true
	case strings.HasPrefix(path, "/is_staged"):
		return true
	}
	return false
}

func (i *record) Save() (row map[string]bigquery.Value, insertID string, err error) {
	row = map[string]bigquery.Value{
		"timestamp":       i.timestamp,
		"hostname":        i.hostname,
		"client_ip":       i.clientIP,
		"http_method":     i.httpMethod,
		"status":          i.status,
		"path":            i.path,
		"body_bytes_sent": i.bodyBytesSent,
		"expected_size":   i.expectedSize,
		"request_time":    i.requestTime,
		"cache":           i.cacheStatus,
	}
	// A unique insert ID can prevent duplicated uploading when the BigQuery client retrys.
	insertID = fmt.Sprintf("%v", row)

	return row, insertID, nil
}
