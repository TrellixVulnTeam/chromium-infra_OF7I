// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package metrics provides common metrics code for Lucifer.
//
// Monitoring configuration is global to the program.  This package sets
// up Stackdriver Trace, BigQuery, and tsmon.
//
// The GOOGLE_APPLICATION_CREDENTIALS environment variable specifies the GCP
// service account credentials for metrics.
//
// A top level trace is set up with the program name, taken from the
// first command argument.
package metrics

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"cloud.google.com/go/bigquery"
	"go.chromium.org/luci/common/tsmon"
	"go.chromium.org/luci/common/tsmon/target"

	"infra/cros/cmd/lucifer/internal/logdog"
)

var programName = filepath.Base(os.Args[0])

// logLabel is used for LabeledLog messages.
const logLabel = "metrics"

// Config describes configuration for Setup.
type Config struct {
	GCPProject string
}

// Client represents all of the monitoring component clients.  The
// client should be closed after use.
type Client struct {
	ctx      context.Context
	logger   logdog.Logger
	bqClient *bigquery.Client
}

// Close closes the monitoring client components.
func (c *Client) Close() error {
	s := c.logger.Step("Flush metrics")
	defer s.Close()
	// Cleanup should be performed in reverse order of Setup.
	if c.bqClient != nil {
		if err := c.bqClient.Close(); err != nil {
			log.Printf("Error closing BigQuery client: %s", err)
		}
	}
	tsmon.Shutdown(c.ctx)
	return nil
}

// Setup configures monitoring based on the given Config.  Make sure
// to close the client after use.  Errors in setting up monitoring
// components will be logged and ignored so the caller does not need
// to worry about stopping the entire program (better to have no
// metrics and think there is an outage than to actually have an
// outage).
func Setup(ctx context.Context, logger logdog.Logger, c Config) (context.Context, *Client) {
	cl := &Client{
		ctx:    ctx,
		logger: logger,
	}
	s := logger.Step("Set up metrics")
	defer s.Close()
	fl := tsmonFlags()
	if err := tsmon.InitializeFromFlags(ctx, &fl); err != nil {
		log.Printf("Skipping tsmon setup: %s", err)
	}
	b, err := bigquery.NewClient(ctx, c.GCPProject)
	if err != nil {
		log.Printf("Skipping BigQuery setup: %s", err)
	} else {
		cl.bqClient = b
	}
	return ctx, cl
}

func tsmonFlags() tsmon.Flags {
	fl := tsmon.NewFlags()
	fl.Flush = tsmon.FlushManual
	fl.Target.SetDefaultsFromHostname()
	fl.Target.TargetType = target.TaskType
	fl.Target.TaskServiceName = programName
	fl.Target.TaskJobName = programName
	return fl
}
