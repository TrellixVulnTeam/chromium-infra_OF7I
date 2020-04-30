// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dumper

import (
	"context"
	"infra/unifiedfleet/app/cron"
	"time"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server"
)

// Options is the dumper server configuration.
type Options struct {
	// CronInterval setups the user-specific cron interval for data dumping
	CronInterval time.Duration
}

// InitServer initializes a purger server.
func InitServer(srv *server.Server, opts Options) {
	srv.RunInBackground("ufs.dumper", func(ctx context.Context) {
		minInterval := 60 * time.Minute
		if opts.CronInterval > 0 {
			minInterval = opts.CronInterval
		}
		run(ctx, minInterval)
	})
}

func run(ctx context.Context, minInterval time.Duration) {
	cron.Run(ctx, minInterval, dumpConfigurations)
}

func dumpConfigurations(ctx context.Context) error {
	logging.Debugf(ctx, "Dumping configuration subsystems")
	return nil
}
