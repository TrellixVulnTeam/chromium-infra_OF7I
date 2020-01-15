// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package update

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"

	"infra/cmd/stable_version2/internal/cmd"
	"infra/cmd/stable_version2/internal/site"
)

// Cmd updates the stable_version2 tool.
var Cmd = &subcommands.Command{
	UsageLine: "update",
	ShortDesc: "update stable_version2 tool",
	LongDesc: `Update stable_version2 tool.
This is just a thin wrapper around CIPD.`,
	CommandRun: func() subcommands.CommandRun {
		c := &updateRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		return c
	},
}

type updateRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

func (c *updateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *updateRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	d, err := cmd.ExecutableDir()
	if err != nil {
		return err
	}
	root, err := cmd.FindCIPDRootDir(d)
	if err != nil {
		return err
	}
	cmd := exec.Command("cipd", "ensure", "-root", root, "-ensure-file", "-")
	cmd.Stdin = strings.NewReader("chromiumos/infra/stable_version2/${platform} latest")
	cmd.Stdout = a.GetOut()
	cmd.Stderr = a.GetErr()
	if err := cmd.Run(); err != nil {
		return err
	}
	fmt.Fprintf(a.GetErr(), "%s: You may need to run stable_version2 login again after the update\n", a.GetName())
	fmt.Fprintf(a.GetErr(), "%s: Run stable_version2 whoami to check login status\n", a.GetName())
	return nil
}
