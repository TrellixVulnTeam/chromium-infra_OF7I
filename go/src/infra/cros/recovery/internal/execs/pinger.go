// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/internal/log"
)

// DefaultPinger returns pinger for current resource name specified per plan.
func (ei *ExecInfo) DefaultPinger() components.Pinger {
	return ei.NewPinger(ei.RunArgs.ResourceName)
}

// NewPinger returns pinger for requested resource.
func (ei *ExecInfo) NewPinger(resource string) components.Pinger {
	pinger := func(ctx context.Context, count int) error {
		log.Debugf(ctx, "Start ping %q %d times", resource, count)
		return ei.RunArgs.Access.Ping(ctx, resource, count)
	}
	return pinger
}
