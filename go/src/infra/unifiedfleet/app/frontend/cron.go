// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	api "infra/unifiedfleet/api/v1/cron"
)

type CronServerImpl struct {
}

func (c *CronServerImpl) TriggerCronJob(context.Context, *api.TriggerCronJobReq) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "Pending implementation")
}
