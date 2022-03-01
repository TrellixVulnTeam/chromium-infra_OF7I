// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package subcmds

import (
	"fmt"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"

	"infra/cros/cmd/satlab/internal/site"
	"infra/libs/skylab/common/heuristics"
)

var IsSatlabRemoteAccessCmd = &subcommands.Command{
	UsageLine: `is-satlab-remote-access`,
	ShortDesc: `Determine whether the current environment is a satlab remote access container.`,
	CommandRun: func() subcommands.CommandRun {
		r := &IsSatlabRemoteAccessRun{}

		return r
	},
}

type IsSatlabRemoteAccessRun struct {
	subcommands.CommandRunBase

	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
}

// Run runs the command, prints the error if there is one, and returns an exit status.
func (c *IsSatlabRemoteAccessRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	isSRA, err := heuristics.LooksLikeSatlabRemoteAccessContainer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot identify environment: %s\n", err)
		return 2
	}
	if isSRA {
		fmt.Printf("%s\n", "yes")
		return 0
	}
	fmt.Printf("%s\n", "no")
	return 1
}
