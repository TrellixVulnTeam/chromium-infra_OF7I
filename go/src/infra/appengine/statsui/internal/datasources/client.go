// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package datasources

import (
	"context"
	"math/big"
	"sort"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"

	"go.chromium.org/luci/grpc/appstatus"

	"infra/appengine/statsui/api"
)

// Client is used to fetch metrics from a given data source.
type Client struct {
	Client *bigquery.Client
	Config *Config
}

func bqToDateArray(dates []string) ([]civil.Date, error) {
	ret := make([]civil.Date, len(dates))
	for i, date := range dates {
		d, err := civil.ParseDate(date)
		if err != nil {
			return nil, err
		}
		ret[i] = d
	}
	return ret, nil
}

func (c *Client) GetMetrics(ctx context.Context, dataSource string, period api.Period, dates, metrics []string) ([]*api.Section, error) {
	if _, exists := c.Config.Sources[dataSource]; !exists {
		return nil, appstatus.Errorf(codes.InvalidArgument, "data source %q not found", dataSource)
	}
	if _, exists := c.Config.Sources[dataSource].Queries[period.String()]; !exists {
		return nil, appstatus.Errorf(codes.InvalidArgument, "period %q not available for data source %q", period, dataSource)
	}
	type LabelValue struct {
		Label string
		Value *big.Rat
	}
	type Row struct {
		Date      civil.Date
		Section   string
		Metric    string
		Value     *big.Rat
		Aggregate []LabelValue
	}
	q := c.Client.Query(c.Config.Sources[dataSource].Queries[period.String()])
	bqDates, err := bqToDateArray(dates)
	if err != nil {
		return nil, err
	}
	q.Parameters = []bigquery.QueryParameter{
		{Name: "dates", Value: bqDates},
		{Name: "metrics", Value: metrics},
	}
	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}
	data := make(map[string]map[string]*api.Metric)
	for {
		var row Row
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		section, ok := data[row.Section]
		if !ok {
			section = make(map[string]*api.Metric)
			data[row.Section] = section
		}
		metric, ok := section[row.Metric]
		if !ok {
			metric = &api.Metric{
				Name: row.Metric,
			}
			section[metric.Name] = metric
		}
		if row.Aggregate != nil {
			for _, val := range row.Aggregate {
				dataSet, ok := metric.Sections[val.Label]
				if !ok {
					dataSet = &api.DataSet{Data: make(map[string]float32)}
					if metric.Sections == nil {
						metric.Sections = make(map[string]*api.DataSet)
					}
					metric.Sections[val.Label] = dataSet
				}
				dataSet.Data[row.Date.String()], _ = val.Value.Float32()
			}
		} else if row.Value != nil {
			if metric.Data == nil {
				metric.Data = &api.DataSet{Data: make(map[string]float32)}
			}
			metric.Data.Data[row.Date.String()], _ = row.Value.Float32()
		}
	}

	sections := make([]*api.Section, 0, len(data))
	for sectionName, m := range data {
		section := &api.Section{
			Name:    sectionName,
			Metrics: make([]*api.Metric, 0, len(m)),
		}
		for _, metric := range m {
			section.Metrics = append(section.Metrics, metric)
		}
		sort.Slice(section.Metrics, func(i, j int) bool {
			return section.Metrics[i].Name < section.Metrics[j].Name
		})
		sections = append(sections, section)
	}
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Name < sections[j].Name
	})

	return sections, nil
}
