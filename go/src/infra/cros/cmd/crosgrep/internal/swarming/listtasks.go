// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/api/iterator"
)

// bqRow is an alias for the type of a bigquery row.
type bqRow = []bigquery.Value

// Limit on the number of swarming tasks that can be retrieved by a single query.
const swarmingTasksLimit = 10000

// This is the pattern for a query that grabs a number of rows corresponding
// to swarming tasks out of bigquery.
//
// Free vars in order: [%d epoch_time_start, %d epoch_time_end, %d limit]
//
// Note that the interval is closed at the beginning and open at the end
//
// Sample query for [1621881767, 1621881769, 1234]:
//
//
// SELECT
//   bot.bot_id,
//   task_id,
//   UNIX_SECONDS(end_time),
//   (SELECT ARRAY_TO_STRING(values, ",") FROM TRS.bot.dimensions WHERE key = "label-model" LIMIT 1)
//     AS model,
// FROM `chromeos-swarming`.swarming.task_results_summary AS TRS
// WHERE
//   REGEXP_CONTAINS(bot.bot_id, r'^(?i)crossk[-]')
//   AND 1621881767 <= UNIX_SECONDS(end_time)
//   AND 1621881769  > UNIX_SECONDS(end_time)
// LIMIT 1234

const listSwarmingTasksInRangePattern = "" +
	"SELECT\n" +
	"  bot.bot_id,\n" +
	"  task_id,\n" +
	"  UNIX_SECONDS(end_time),\n" +
	"  (SELECT ARRAY_TO_STRING(values, ',') FROM TRS.bot.dimensions WHERE key = 'label-model' LIMIT 1) AS model,\n" +
	"FROM `chromeos-swarming`.swarming.task_results_summary AS TRS\n" +
	"WHERE\n" +
	"  REGEXP_CONTAINS(bot.bot_id, r'^(?i)crossk[-]')\n" +
	"  AND %d <= UNIX_SECONDS(end_time)\n" +
	"  AND %d  > UNIX_SECONDS(end_time)\n" +
	"LIMIT %d\n" +
	""

func formatListSwarmingTasksInRange(start int64, stop int64, limit int) string {
	return fmt.Sprintf(listSwarmingTasksInRangePattern, start, stop, limit)
}

// This is the pattern for a query that grabs a number of rows corresponding
// to swarming tasks out of bigquery.
//
// Free vars in order: [%d epoch_time_start, %d epoch_time_end, %s model, %d limit]
//
// Note that the interval is closed at the beginning and open at the end
//
// Sample query for [1621881767, 1621881769, "FAKE-MODEL", 1234]:
//
//
// SELECT
//   bot.bot_id,
//   task_id,
//   UNIX_SECONDS(end_time),
//   (SELECT ARRAY_TO_STRING(values, ",") FROM TRS.bot.dimensions WHERE key = "label-model" LIMIT 1)
//     AS model,
// FROM `chromeos-swarming`.swarming.task_results_summary AS TRS
// WHERE
//   REGEXP_CONTAINS(bot.bot_id, r'^(?i)crossk[-]')
//   AND 1621881767 <= UNIX_SECONDS(end_time)
//   AND 1621881769  > UNIX_SECONDS(end_time)
//   AND (
//     SELECT SUM(IF("FAKE-MODEL" IN UNNEST(values), 1, 0))
//     FROM TRS.bot.dimensions
//     WHERE key = 'label-model'
//     LIMIT 1
//   ) > 0
// LIMIT 1234

const listSwarmingTasksWithModelInRangePattern = "" +
	"SELECT\n" +
	"  bot.bot_id,\n" +
	"  task_id,\n" +
	"  UNIX_SECONDS(end_time),\n" +
	"  (SELECT ARRAY_TO_STRING(values, ',') FROM TRS.bot.dimensions WHERE key = 'label-model' LIMIT 1) AS model,\n" +
	"FROM `chromeos-swarming`.swarming.task_results_summary AS TRS\n" +
	"WHERE\n" +
	"  REGEXP_CONTAINS(bot.bot_id, r'^(?i)crossk[-]')\n" +
	"  AND %d <= UNIX_SECONDS(end_time)\n" +
	"  AND %d  > UNIX_SECONDS(end_time)\n" +
	"  AND (\n" +
	"    SELECT SUM(IF(%q IN UNNEST(values), 1, 0))\n" +
	"    FROM TRS.bot.dimensions\n" +
	"    WHERE key = 'label-model'\n" +
	"    LIMIT 1\n" +
	"  ) > 0\n" +
	"LIMIT %d\n" +
	""

func formatListSwarmingTasksWithModelInRange(start int64, stop int64, model string, limit int) string {
	return fmt.Sprintf(listSwarmingTasksWithModelInRangePattern, start, stop, model, limit)
}

// MakeSwarmingTasksInRangeQuery makes a list of all swarming tasks in a particular time range
// up to a specified limit.
//
// limit:      limit on the number of tasks to return
// rangeStart: the time to begin the search
// rangeStop:  the time to end the search
// model:      the model to search
//
func MakeSwarmingTasksInRangeQuery(limit int, rangeStart int64, rangeStop int64, model string) string {
	if limit == 0 {
		limit = swarmingTasksLimit
	}
	// Default start time is one hour before the present.
	if rangeStart == 0 {
		rangeStop = time.Now().Unix() - 3600
	}
	// The default stop time is one second after the present.
	// It is this value rather than simply the present because
	// in the query the intervals are all of the form [start, stop).
	if rangeStop == 0 {
		rangeStop = time.Now().Unix() + 1
	}
	if model == "" {
		return formatListSwarmingTasksInRange(rangeStart, rangeStop, limit)
	}
	return formatListSwarmingTasksWithModelInRange(rangeStart, rangeStop, model, limit)
}

// getRowIterator returns a row iterator from a sql query.
func getRowIterator(ctx context.Context, client *bigquery.Client, sql string) (*bigquery.RowIterator, error) {
	logging.Debugf(ctx, "GetRowIterator %20s\n", strings.ReplaceAll(sql, "\n", "\t"))
	q := client.Query(sql)
	it, err := q.Read(ctx)
	return it, errors.Annotate(err, "get iterator").Err()
}

// getSwarmingTasks gets a list of swarming tasks in an arbitrary manner.
func getSwarmingTasks(ctx context.Context, client *bigquery.Client, model string) (*bigquery.RowIterator, error) {
	logging.Debugf(ctx, "GetSwarmingTasks\n")
	if client == nil {
		panic("client cannot be nil")
	}
	it, err := getRowIterator(ctx, client, MakeSwarmingTasksInRangeQuery(0, 0, 0, model))
	return it, errors.Annotate(err, "get iterator").Err()
}

// ExtractValues extracts all the values from a sql query result set.
func ExtractValues(ctx context.Context, client *bigquery.Client, model string) ([]bqRow, error) {
	if client == nil {
		panic("client cannot be nil")
	}
	var out []bqRow
	it, err := getSwarmingTasks(ctx, client, model)
	if err != nil {
		return nil, errors.Annotate(err, "getting tasks").Err()
	}
	// TODO(gregorynisbet): Consider adding a bound here if possible.
	for {
		logging.Debugf(ctx, "Reading one item\n")
		var item bqRow
		err := it.Next(&item)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, errors.Annotate(err, "reading value").Err()
		}
		out = append(out, item)
	}
	return out, nil
}
