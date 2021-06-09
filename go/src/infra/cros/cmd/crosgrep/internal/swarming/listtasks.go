// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/api/iterator"
)

// SwarmingTasksLimit is the limit on the number of swarming tasks that can be retrieved by a single query.
const swarmingTasksLimit = 10000

// MakeSwarmingTasksInRangeQuery makes a list of all swarming tasks in a particular time range
// up to a specified limit.
//
// limit:      limit on the number of tasks to return
// rangeStart: the time to begin the search
// rangeStop:  the time to end the search
// model:      the model to search
//
func MakeSwarmingTasksInRangeQuery(limit int, rangeStart int64, rangeStop int64, model string) (string, error) {
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
	sql, err := templateToString(
		tmplTasksQuery,
		&TasksQueryParams{
			Model:     model,
			StartTime: rangeStart,
			EndTime:   rangeStop,
			Limit:     limit,
		},
	)
	if err != nil {
		return "", err
	}
	return sql, nil
}

// GetSwarmingTasks gets a list of swarming tasks in an arbitrary manner.
func getSwarmingTasks(ctx context.Context, client *bigquery.Client, model string) (*bigquery.RowIterator, error) {
	logging.Debugf(ctx, "GetSwarmingTasks\n")
	if client == nil {
		panic("client cannot be nil")
	}
	sql, err := MakeSwarmingTasksInRangeQuery(0, 0, 0, model)
	if err != nil {
		return nil, errors.Annotate(err, "make query").Err()
	}
	it, err := getRowIterator(ctx, client, sql)
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
