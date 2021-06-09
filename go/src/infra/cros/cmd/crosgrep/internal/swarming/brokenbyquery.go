// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/api/iterator"
)

// DefaultOffsetFromPresent is the amount of time that we look into the past
// for a possible task by default.
// We set it to ten days in seconds.
const defaultOffsetFromPresent = -10 * 24 * 3600

// BrokenByParams are all the parameters necessary to determine which task broke a DUT when
type BrokenByParams struct {
	BotID     string
	StartTime int64
	EndTime   int64
}

// TmplBrokenBy is a SQL query that shows the last task to run successfully on a given hostname and when it ended.
var tmplBrokenBy = templateOrPanic(
	"brokenBy",
	tmplPreamble+
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

// GetBrokenBy takes BigQuery client, a botID, a time range and returns a result set
// where each result is a dictionary mapping the column name to the value.
func GetBrokenBy(ctx context.Context, client *bigquery.Client, botID string, rangeStart int64, rangeStop int64) ([]bqRow, error) {
	if client == nil {
		panic("client cannot be nil")
	}
	if botID == "" {
		return nil, errors.New("GetBrokenBy: empty botID")
	}
	botID = toBotName(botID)
	if rangeStart == 0 {
		rangeStart = time.Now().Unix() + defaultOffsetFromPresent
	}
	if rangeStop == 0 {
		rangeStop = time.Now().Unix() + 1
	}
	sql, err := templateToString(
		tmplBrokenBy,
		&BrokenByParams{
			BotID:     botID,
			StartTime: rangeStart,
			EndTime:   rangeStop,
		},
	)
	if err != nil {
		return nil, err
	}
	it, err := getRowIterator(ctx, client, sql)
	if err != nil {
		return nil, err
	}
	var out []bqRow
	for {
		var item bqRow
		err := it.Next(&item)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, errors.Annotate(err, "advancing to next record").Err()
		}
		out = append(out, item)
	}
	return out, nil
}
