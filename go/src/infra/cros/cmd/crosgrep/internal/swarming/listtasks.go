// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"context"
	"strings"

	"cloud.google.com/go/bigquery"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/api/iterator"
)

// bqRow is the type of a bigquery row.
type bqRow []bigquery.Value

// listAllSwarmingTasks lists all the swarming tasks.
// TODO(gregorynisbet): replace with call to library that generates queries.
const listAllSwarmingTasks = `
SELECT task_id,
FROM ` + "`" + `chromeos-swarming` + "`" + `.swarming.task_results_summary
WHERE REGEXP_CONTAINS(bot.bot_id, r"^(?i)crossk[-]")
LIMIT 1000
`

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
	it, err := getRowIterator(ctx, client, listAllSwarmingTasks)
	return it, errors.Annotate(err, "get iterator").Err()
}

// ExtractNValues extracts N values from a sql query.
func ExtractNValues(ctx context.Context, client *bigquery.Client, limit int) ([]bqRow, error) {
	if client == nil {
		panic("client cannot be nil")
	}
	if limit <= 0 {
		panic("Invalid non-positive argument to ExtractNValues")
	}
	var out []bqRow
	it, err := getSwarmingTasks(ctx, client)
	if err != nil {
		return nil, errors.Annotate(err, "getting tasks").Err()
	}
	for i := 1; i <= limit; i++ {
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
