// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"infra/cros/karte/internal/site"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
)

// Backfill backfills a single swarming task to Karte.
var Backfill = &subcommands.Command{
	UsageLine: `backfill`,
	ShortDesc: `backfill a swarming task`,
	LongDesc:  `Take a swarming task ID and backfill its device health profile.`,
	CommandRun: func() subcommands.CommandRun {
		r := &backfillRun{}
		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		// TODO(gregorynisbet): add envFlags
		return r
	},
}

// BackfillRun stores the arguments for the backfill command.
type backfillRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

// Run runs the check-server command and returns an exit status.
func (c *backfillRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	// Not implemented.
	return 1
}
