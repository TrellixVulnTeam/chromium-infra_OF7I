// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package metrics

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"

	"infra/cros/cmd/lucifer/internal/event"
	"infra/cros/cmd/lucifer/internal/logdog"
)

// A BQClient emits BigQuery events.
type BQClient struct {
	logger logdog.Logger
	// bqClient may be nil.
	bqClient *bigquery.Client
}

// BigQuery returns a dispatcher for BigQuery actions.
func (c *Client) BigQuery() BQClient {
	return BQClient{
		logger:   c.logger,
		bqClient: c.bqClient,
	}
}

// HostStatusEvent inserts a host status event in BigQuery.
func (c BQClient) HostStatusEvent(ctx context.Context, hostname string, e event.Event) {
	if c.bqClient == nil {
		return
	}
	bqc := c.bqClient
	u := bqc.Dataset("staging").Table("dut_status").Uploader()
	type Related struct {
		Type string
		ID   int
	}
	type DUTStatus struct {
		Time     time.Time
		Hostname string
		Status   string
		Related  []Related
	}
	s := DUTStatus{
		Time:     time.Now(),
		Hostname: hostname,
		Status:   string(e),
	}
	if err := u.Put(ctx, s); err != nil {
		switch err := err.(type) {
		case bigquery.PutMultiError:
			for _, err := range err {
				msg := fmt.Sprintf("Error recording host status in BigQuery: %s", err.Error())
				c.logger.LabeledLog(logLabel, msg)
			}
		default:
			msg := fmt.Sprintf("Error recording host status in BigQuery: %s", err.Error())
			c.logger.LabeledLog(logLabel, msg)

		}
	}
}
