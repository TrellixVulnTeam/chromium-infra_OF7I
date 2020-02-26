// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package meta

import (
	"context"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"

	"infra/cmd/labtool/site"
	"infra/libs/cros/cipd"
)

// Version subcommand: Version labtool.
var Version = &subcommands.Command{
	UsageLine: "version",
	ShortDesc: "print labtool version",
	LongDesc:  "Print labtool version.",
	CommandRun: func() subcommands.CommandRun {
		c := &versionRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		return c
	},
}

type versionRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

func (c *versionRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *versionRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	p, err := cipd.FindPackage("labtool", "chromiumos/infra/labtool/")
	if err != nil {
		return err
	}
	ctx := context.Background()
	d, err := cipd.DescribePackage(ctx, p.Package, p.Pin.InstanceID)
	if err != nil {
		return err
	}

	fmt.Printf("Package:\t%s\n", p.Package)
	fmt.Printf("Version:\t%s\n", p.Pin.InstanceID)
	fmt.Printf("Updated:\t%s\n", d.RegisteredTs)
	fmt.Printf("Tracking:\t%s\n", p.Tracking)
	return nil
}
