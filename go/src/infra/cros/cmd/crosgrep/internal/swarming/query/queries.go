// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package query

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"
	"go.chromium.org/luci/common/errors"
)

// BrokenByParams are all the parameters necessary to determine which task broke a DUT when
type BrokenByParams struct {
	BotID     string
	StartTime int64
	EndTime   int64
}

// RunBrokenBy takes a BigQuery client and parameters and returns a result set.
func RunBrokenBy(ctx context.Context, client *bigquery.Client, params *BrokenByParams) (*bigquery.RowIterator, error) {
	now := time.Now().Unix()
	if params.BotID == "" {
		return nil, errors.New("BotID cannot be empty")
	}
	if params.StartTime == 0 {
		// Set the default search range to one hour before the present. This choice
		// empirically leads to fast queries.
		params.StartTime = now - 3600
	}
	if params.EndTime == 0 {
		params.EndTime = now + 1
	}
	sql, err := instantiateSQLQuery(ctx, brokenByTemplate, params)
	if err != nil {
		return nil, err
	}
	it, err := RunSQL(ctx, client, sql)
	if err != nil {
		return nil, errors.Annotate(err, "run broken by").Err()
	}
	return it, nil
}

// BrokenByTemplate is a fixed query that finds the last successful task to execute on a given host.
var brokenByTemplate = mustMakeTemplate(
	"brokenBy",
	`
SELECT
  TRS.bot.bot_id,
  BUILDS.id AS bbid,
  TRS.task_id,
  UNIX_SECONDS(TRS.end_time),
  (SELECT ARRAY_TO_STRING(values, ",") FROM TRS.bot.dimensions WHERE key = "label-model" LIMIT 1)
    AS model,
FROM {{$tick}}chromeos-swarming.swarming.task_results_summary{{$tick}} AS TRS
  LEFT OUTER JOIN
    {{$tick}}cr-buildbucket.chromeos.builds{{$tick}} AS BUILDS
    ON TRS.task_id = BUILDS.infra.swarming.task_id
WHERE
  REGEXP_CONTAINS(TRS.bot.bot_id, r'^(?i)crossk[-]')
  AND TRS.exit_code = 0
  AND {{.BotID | printf "%q"}} = TRS.bot.bot_id
  AND {{.StartTime | printf "%d"}} <= UNIX_SECONDS(TRS.end_time)
  AND {{.EndTime | printf "%d"}}  > UNIX_SECONDS(TRS.end_time)
  AND {{.StartTime | printf "%d"}} <= UNIX_SECONDS(BUILDS.end_time) + 15000
  AND {{.EndTime | printf "%d"}}  > UNIX_SECONDS(BUILDS.end_time) - 15000
ORDER BY TRS.end_time DESC
LIMIT 1
`,
)

// BuildBucketSafetyMarginSeconds is the number of seconds to back off.
// In our queries, the time range applies to the end_time of the swarming task and
// not the buildbucket task. Therefore, in order to make sure that we don't miss
// any entries for buildbucket associated with swarming records, we use a fixed
// "margin of error". For details, see the example queries.
//
// This constant exists to guard against potential data discrepancies in buildbucket
// data. The reason that this constant exists at all is so that we can still filter
// the buildbucket table before joining it with the swarming results table, which
// improves the speed of the query.
//
// This constant ensures that we pick up buildbucket records corresponding to entries
// that ended after the end of the window and began before the start of the window.
// This is necessary because the filter time applies to the swarming task time alone,
// so we need to make sure that we look at a big enough range of buildbucket tasks to
// ensure that we always find the buildbucket task associated with a swarming task.
const buildBucketSafetyMarginSeconds = 15000

// SwarmingTasksLimit is a reasonable upper limit on the number of
// swarming tasks returned by a find task query.
const swarmingTasksLimit = 10000

// TaskQueryParams are all the params necessary to construct a task query
type TaskQueryParams struct {
	// Params.Model may be empty or non-empty. An empty string as the model means that all
	// models are permitted.
	Model     string
	StartTime int64
	EndTime   int64
	Limit     int
	// BuildBucketSafetyMargin is the number of seconds to look before and after
	// the given time range in order to be sure to include all buildbucket records.
	// For more details, see the documentation for buildBucketSafetyMarginSeconds.
	BuildBucketSafetyMargin int64
}

// RunTaskQuery takes a BigQuery client and parameters and returns a result set.
func RunTaskQuery(ctx context.Context, client *bigquery.Client, params *TaskQueryParams) (*bigquery.RowIterator, error) {
	if params.StartTime == 0 {
		// Set the default search range to one hour before the present. This choice
		// empirically leads to fast queries.
		params.StartTime = time.Now().Unix() - 3600
	}
	if params.EndTime == 0 {
		params.EndTime = time.Now().Unix() + 1
	}
	if params.Limit == 0 {
		params.Limit = swarmingTasksLimit
	}
	if params.BuildBucketSafetyMargin == 0 {
		params.BuildBucketSafetyMargin = buildBucketSafetyMarginSeconds
	}
	sql, err := instantiateSQLQuery(ctx, taskQueryTemplate, params)
	if err != nil {
		return nil, err
	}
	it, err := RunSQL(ctx, client, sql)
	if err != nil {
		return nil, errors.Annotate(err, "run tasks query").Err()
	}
	return it, nil
}

// TaskQueryTemplate is a query factory based on a SQL template that extracts information about swarming tasks
// and their corresponding buildbucket tasks.
var taskQueryTemplate = mustMakeTemplate(
	"taskQuery",
	`
SELECT
  TRS.bot.bot_id,
  BUILDS.id AS bbid,
  JSON_EXTRACT_SCALAR(BUILDS.output.properties, r"$.compressed_result") AS bb_output_properties,
  TRS.task_id,
  UNIX_SECONDS(TRS.end_time) AS end_time,
  (SELECT ARRAY_TO_STRING(values, ",") FROM TRS.bot.dimensions WHERE key = "label-model" LIMIT 1)
    AS model,
FROM {{$tick}}chromeos-swarming.swarming.task_results_summary{{$tick}} AS TRS
  LEFT OUTER JOIN
    {{$tick}}cr-buildbucket.chromeos.builds{{$tick}} AS BUILDS
    ON TRS.task_id = BUILDS.infra.swarming.task_id
WHERE
  REGEXP_CONTAINS(TRS.bot.bot_id, r'^(?i)crossk[-]')
  AND {{.StartTime | printf "%d"}} <= UNIX_SECONDS(TRS.end_time)
  AND {{.EndTime | printf "%d"}}  > UNIX_SECONDS(TRS.end_time)
  AND {{.StartTime | printf "%d"}} <= UNIX_SECONDS(BUILDS.end_time) + 15000
  AND {{.EndTime | printf "%d"}}  > UNIX_SECONDS(BUILDS.end_time) - 15000
  AND (
    ({{.Model | printf "%q"}} = '') OR
    (
      SELECT SUM(IF({{.Model | printf "%q"}} IN UNNEST(values), 1, 0))
      FROM TRS.bot.dimensions
      WHERE key = 'label-model'
      LIMIT 1
    ) > 0
 )
LIMIT {{.Limit | printf "%d"}}
`,
)

var randTaskTemplate = mustMakeTemplate(
	"randTask",
	`
SELECT
  TRS.task_id AS task_id,
  TRS.request.name AS name,
  TRS.exit_code AS exit_code,
FROM {{$tick}}chromeos-swarming.swarming.task_results_summary{{$tick}} AS TRS
  WHERE {{.StartTime | printf "%d"}} <= UNIX_SECONDS(TRS.end_time)
  AND {{.EndTime | printf "%d"}}  > UNIX_SECONDS(TRS.end_time)
ORDER BY RAND()
LIMIT 1
`,
)

// RandTaskParams are all the params necessary to fetch a random task.
type RandTaskParams struct {
	Model                   string
	StartTime               int64
	EndTime                 int64
	Limit                   int
	BuildBucketSafetyMargin int64
}

// RunRandTaskQuery takes a BigQuery client and parameters and returns a randomly chosen task
// fitting the requirements, its request name and its exit status.
func RunRandTaskQuery(ctx context.Context, client *bigquery.Client, params *RandTaskParams) (*bigquery.RowIterator, error) {
	// Params.Model may be empty or non-empty. A non-empty model means that all
	// models are permitted.
	if params.StartTime == 0 {
		// Set the default search range to one hour before the present. This choice
		// empirically leads to fast queries.
		params.StartTime = time.Now().Unix() - 3600
	}
	if params.EndTime == 0 {
		params.EndTime = time.Now().Unix() + 1
	}
	if params.Limit == 0 {
		params.Limit = swarmingTasksLimit
	}
	if params.BuildBucketSafetyMargin == 0 {
		params.BuildBucketSafetyMargin = buildBucketSafetyMarginSeconds
	}
	sql, err := instantiateSQLQuery(ctx, randTaskTemplate, params)
	if err != nil {
		return nil, err
	}
	it, err := RunSQL(ctx, client, sql)
	if err != nil {
		return nil, errors.Annotate(err, "run tasks query").Err()
	}
	return it, nil
}

// GetStatusLogParams are all the params necessary to construct a query to get the status
// logs for a single swarming task.
type GetStatusLogParams struct {
	SwarmingTaskID string
}

// GetStatusLogQuery is a query that gets the buildbucket ID and result proto for a single
// swarming task.
// We do not need to join the task_results_summary table because the table of buildbucket builds
// already contains the swarming ID.
// In the event that no swarming task ID is specified, yield the status log query for an arbitrary
// swarming query.
var GetStatusLogQuery = mustMakeTemplate(
	"getStatusLog",
	`
SELECT
  BUILDS.infra.swarming.task_id AS swarming_id,
  BUILDS.id AS bbid,
  JSON_EXTRACT_SCALAR(BUILDS.output.properties, r"$.compressed_result") AS bb_output_properties,
FROM
  {{$tick}}cr-buildbucket.chromeos.builds{{$tick}} AS BUILDS
WHERE
  (
    BUILDS.infra.swarming.task_id = {{.SwarmingTaskID | printf "%q"}}
    OR "" = {{.SwarmingTaskID | printf "%q"}}
  )
LIMIT 1
`,
)

// RunStatusLogQuery takes a bigquery client and parameters and returns a result set.
func RunStatusLogQuery(ctx context.Context, client *bigquery.Client, params *GetStatusLogParams) (*bigquery.RowIterator, error) {
	sql, err := instantiateSQLQuery(ctx, GetStatusLogQuery, params)
	if err != nil {
		return nil, err
	}
	it, err := RunSQL(ctx, client, sql)
	if err != nil {
		return nil, errors.Annotate(err, "run tasks query").Err()
	}
	return it, nil
}
