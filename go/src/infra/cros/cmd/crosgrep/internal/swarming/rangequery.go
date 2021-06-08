// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

// buildBucketSafetyMarginSeconds is the number of seconds to back off.
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

// TasksQueryParams are all the params necessary to construct a task query
type TasksQueryParams struct {
	Model                   string
	StartTime               int64
	EndTime                 int64
	Limit                   int
	BuildBucketSafetyMargin int64
}

// TmplTasksQuery is a SQL template that extracts information about swarming tasks
// and their corresponding buildbucket tasks.
var tmplTasksQuery = templateOrPanic(
	"tasksQuery",
	tmplPreamble+
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
