// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/api/iterator"
)

// bqRow is an alias for the type of a bigquery row.
type bqRow = []bigquery.Value

// Limit on the number of swarming tasks that can be retrieved by a single query.
const swarmingTasksLimit = 1000

// This is the pattern for a query that grabs a number of rows corresponding
// to swarming tasks out of bigquery.
//
// Free vars in order: [%d limit]
//
// Sample query for [1000]:
//
//   SELECT task_id,
//   FROM `chromeos-swarming`.swarming.task_results_summary
//   WHERE REGEXP_CONTAINS(bot.bot_id, r'^(?i)crossk[-]')
//   ORDER BY end_time DESC
//   LIMIT 1000
//
const listAllSwarmingTasksPattern = "" +
	"SELECT task_id,\n" +
	"FROM `chromeos-swarming`.swarming.task_results_summary\n" +
	"WHERE REGEXP_CONTAINS(bot.bot_id, r'(?i)^crossk[-]')\n" +
	"ORDER BY end_time DESC\n" +
	"LIMIT %d\n"

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
	it, err := getRowIterator(ctx, client, fmt.Sprintf(listAllSwarmingTasksPattern, swarmingTasksLimit))
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
