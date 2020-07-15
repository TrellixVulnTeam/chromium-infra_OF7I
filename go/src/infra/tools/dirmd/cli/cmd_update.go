// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/tools/dirmd/cli/updater"
)

func cmdChromiumUpdate() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `chromium-update`,
		ShortDesc: "INTERNAL tool. Do not use it unless you know what you are doing.",
		Advanced:  true,
		CommandRun: func() subcommands.CommandRun {
			r := &chromiumUpdateRun{}
			r.Flags.StringVar(&r.OutDir, "out-dir", "", "Path to a directory where to write output files")
			r.Flags.StringVar(&r.ChromiumCheckout, "chromium-checkout", "", "Path to the chromium/src.git checkout")
			r.Flags.BoolVar(&r.Prod, "prod", false, "Whether to make production side effects")
			return r
		},
	}
}

type chromiumUpdateRun struct {
	baseCommandRun
	updater.Updater
}

func (r *chromiumUpdateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)

	if r.ChromiumCheckout == "" {
		r.done(ctx, errors.Reason("-chromium-checkout is required").Err())
	}

	return r.done(ctx, r.Updater.Run(ctx))
}
