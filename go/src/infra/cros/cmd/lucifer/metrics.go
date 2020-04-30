// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"

	"infra/cros/cmd/lucifer/internal/api"
	"infra/cros/cmd/lucifer/internal/event"
)

func sendHostStatus(ctx context.Context, ac *api.Client, hosts []string, e event.Event) {
	for _, host := range hosts {
		event.SendWithMsg(e, host)
		ac.BigQuery().HostStatusEvent(ctx, host, e)
	}
}
