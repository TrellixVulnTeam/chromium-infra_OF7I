// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"io/ioutil"

	"cloud.google.com/go/bigquery"
	"google.golang.org/grpc/codes"

	"go.chromium.org/luci/config/server/cfgmodule"
	"go.chromium.org/luci/grpc/appstatus"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"

	"infra/appengine/statsui/api"
	"infra/appengine/statsui/internal/datasources"
)

func main() {
	modules := []module.Module{
		cfgmodule.NewModuleFromFlags(),
		gaeemulation.NewModuleFromFlags(),
	}
	server.Main(nil, modules, func(srv *server.Server) error {
		dsClient, err := setupDataSourceClient(srv.Context)
		if err != nil {
			return err
		}
		stats := &statsServer{
			DataSources: dsClient,
		}
		api.RegisterStatsServer(srv.PRPC, stats)
		return nil
	})
}

func setupDataSourceClient(ctx context.Context) (*datasources.Client, error) {
	yaml, err := ioutil.ReadFile("datasources.yaml")
	if err != nil {
		return nil, err
	}
	config, err := datasources.UnmarshallConfig(yaml)
	if err != nil {
		return nil, err
	}
	bqClient, err := bigquery.NewClient(ctx, "chrome-trooper-analytics")
	if err != nil {
		return nil, err
	}
	return &datasources.Client{
		Client: bqClient,
		Config: config,
	}, nil
}

type statsServer struct {
	DataSources *datasources.Client
}

func (s *statsServer) FetchMetrics(ctx context.Context, req *api.FetchMetricsRequest) (*api.FetchMetricsResponse, error) {
	if len(req.Metrics) == 0 {
		return nil, appstatus.Errorf(codes.InvalidArgument, "no metrics specified")
	}
	sections, err := s.DataSources.GetMetrics(ctx, req.DataSource, req.Period, req.Dates, req.Metrics)
	if err != nil {
		return nil, err
	}
	return &api.FetchMetricsResponse{
		Sections: sections,
	}, nil
}
