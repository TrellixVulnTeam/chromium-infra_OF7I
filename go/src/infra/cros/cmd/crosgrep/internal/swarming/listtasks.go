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
// SELECT bot.bot_id, task_id, UNIX_SECONDS(end_time),
// FROM `chromeos-swarming`.swarming.task_results_summary
// WHERE
//   REGEXP_CONTAINS(bot.bot_id, r'^(?i)crossk[-]')
//     AND 1621881767 <= UNIX_SECONDS(end_time)
//     AND 1621881769  > UNIX_SECONDS(end_time)
//     LIMIT 1234
//
const listSwarmingTasksInRangePattern = "" +
	"SELECT bot.bot_id, task_id, UNIX_SECONDS(end_time),\n" +
	"FROM `chromeos-swarming`.swarming.task_results_summary\n" +
	"WHERE\n" +
	"    REGEXP_CONTAINS(bot.bot_id, r'^(?i)crossk[-]')\n" +
	"    AND %d <= UNIX_SECONDS(end_time)" +
	"    AND %d  > UNIX_SECONDS(end_time)" +
	"LIMIT %d\n"

// MakeSwarmingTasksInRangeQuery makes a list of all swarming tasks in a particular time range
// up to a specified limit.
//
// limit:      limit on the number of tasks to return
// rangeStart: the time to begin the search
// rangeStop:  the time to end the search
//
func MakeSwarmingTasksInRangeQuery(limit int, rangeStart int64, rangeStop int64) string {
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
	// Make sure that the variables below are provided in the correct order for the pattern.
	// If the query changes, the order of its "free variables" may change as well.
	return fmt.Sprintf(listSwarmingTasksInRangePattern, rangeStart, rangeStop, limit)
}

// getRowIterator returns a row iterator from a sql query.
func getRowIterator(ctx context.Context, client *bigquery.Client, sql string) (*bigquery.RowIterator, error) {
	logging.Debugf(ctx, "GetRowIterator %20s\n", strings.ReplaceAll(sql, "\n", "\t"))
	q := client.Query(sql)
	it, err := q.Read(ctx)
	return it, errors.Annotate(err, "get iterator").Err()
}

// getSwarmingTasks gets a list of swarming tasks in an arbitrary manner.
func getSwarmingTasks(ctx context.Context, client *bigquery.Client) (*bigquery.RowIterator, error) {
	logging.Debugf(ctx, "GetSwarmingTasks\n")
	if client == nil {
		panic("client cannot be nil")
	}
	it, err := getRowIterator(ctx, client, MakeSwarmingTasksInRangeQuery(0, 0, 0))
	return it, errors.Annotate(err, "get iterator").Err()
}

// ExtractValues extracts all the values from a sql query result set.
func ExtractValues(ctx context.Context, client *bigquery.Client) ([]bqRow, error) {
	if client == nil {
		panic("client cannot be nil")
	}
	var out []bqRow
	it, err := getSwarmingTasks(ctx, client)
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
