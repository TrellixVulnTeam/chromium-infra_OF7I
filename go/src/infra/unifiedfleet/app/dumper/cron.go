// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dumper

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	api "infra/unifiedfleet/api/v1/cron"
)

type CronServerImpl struct {
}

// TriggerCronJob triggers the given cron job to run. Fails if the Job is already running.
func (c *CronServerImpl) TriggerCronJob(ctx context.Context, job *api.TriggerCronJobReq) (*empty.Empty, error) {
	if err := job.Validate(); err != nil {
		return nil, err
	}
	return &empty.Empty{}, TriggerJob(job.JobName)
}
