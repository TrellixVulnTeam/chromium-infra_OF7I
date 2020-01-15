// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package meta

import (
	"context"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"infra/cmd/stable_version2/internal/site"
)

// Version is a command that prints version of stable_version2 tool.
var Version = &subcommands.Command{
	UsageLine: "version",
	ShortDesc: "print stable_version2 tool version",
	LongDesc:  "Print stable_version2 tool version.",
	CommandRun: func() subcommands.CommandRun {
		c := &command{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		return c
	},
}

type command struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

func (c *command) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}
func (c *command) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	p, err := findStableVersion2Package()
	if err != nil {
		return err
	}
	ctx := context.Background()
	d, err := describe(ctx, p.Package, p.Pin.InstanceID)
	if err != nil {
		return err
	}
	fmt.Printf("Package:\t%s\n", p.Package)
	fmt.Printf("Version:\t%s\n", p.Pin.InstanceID)
	fmt.Printf("Updated:\t%s\n", d.RegisteredTs)
	fmt.Printf("Tracking:\t%s\n", p.Tracking)
	return nil
}
