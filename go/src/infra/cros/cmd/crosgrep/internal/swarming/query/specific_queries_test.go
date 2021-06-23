// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package query

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBrokenBy(t *testing.T) {
	bg := context.Background()
	expected := mustExpandTick(`
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
  AND "FAKE-BOT" = TRS.bot.bot_id
  AND 1 <= UNIX_SECONDS(TRS.end_time)
  AND 2  > UNIX_SECONDS(TRS.end_time)
  AND 1 <= UNIX_SECONDS(BUILDS.end_time) + 15000
  AND 2  > UNIX_SECONDS(BUILDS.end_time) - 15000
ORDER BY TRS.end_time DESC
LIMIT 1
`)
	actual, err := instantiateSQLQuery(
		bg,
		brokenByTemplate,
		&BrokenByParams{
			StartTime: 1,
			EndTime:   2,
			BotID:     "FAKE-BOT",
		},
	)
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRunTasksQuery(t *testing.T) {
	bg := context.Background()
	expected := mustExpandTick(`
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
  AND 1 <= UNIX_SECONDS(TRS.end_time)
  AND 2  > UNIX_SECONDS(TRS.end_time)
  AND 1 <= UNIX_SECONDS(BUILDS.end_time) + 15000
  AND 2  > UNIX_SECONDS(BUILDS.end_time) - 15000
  AND (
    ("" = '') OR
    (
      SELECT SUM(IF("" IN UNNEST(values), 1, 0))
      FROM TRS.bot.dimensions
      WHERE key = 'label-model'
      LIMIT 1
    ) > 0
 )
LIMIT 10000
`)
	actual, err := instantiateSQLQuery(
		bg,
		taskQueryTemplate,
		&TaskQueryParams{
			StartTime:               1,
			EndTime:                 2,
			Model:                   "",
			Limit:                   10000,
			BuildBucketSafetyMargin: 15000,
		},
	)
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// TestRunStatusLogQuery tests that the status log query expands correctly.
func TestRunStatusLogQuery(t *testing.T) {
	t.Parallel()
	bg := context.Background()
	expected := mustExpandTick(`
SELECT
  BUILDS.infra.swarming.task_id AS swarming_id,
  BUILDS.id AS bbid,
  JSON_EXTRACT_SCALAR(BUILDS.output.properties, r"$.compressed_result") AS bb_output_properties,
FROM
  {{$tick}}cr-buildbucket.chromeos.builds{{$tick}} AS BUILDS
WHERE
  (
    BUILDS.infra.swarming.task_id = "AAAAA"
    OR "" = "AAAAA"
  )
LIMIT 1
`)
	actual, err := instantiateSQLQuery(
		bg,
		GetStatusLogQuery,
		&GetStatusLogParams{
			SwarmingTaskID: "AAAAA",
		},
	)
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
