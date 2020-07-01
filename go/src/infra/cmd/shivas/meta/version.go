// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package meta

import (
	"context"
	"fmt"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"

	"infra/cmd/shivas/site"
	"infra/libs/cros/cipd"
)

// VersionNumber is the version number for the tool. It follows the Semantic
// Versioning Specification (http://semver.org) and the format is:
// "MAJOR.MINOR.0+BUILD_TIME".
// We can ignore the PATCH part (i.e. it's always 0) to make the maintenance
// work easier.
// We can also print out the build time (e.g. 20060102150405) as the METADATA
// when show version to users.
var VersionNumber = fmt.Sprintf("%d.%d.%d", Major, Minor, Patch)

// Major is the Major version number
const Major = 2

// Minor is the Minor version number
const Minor = 0

// Patch is the PAtch version number
const Patch = 0

// ClientVersion used as a key in metadata within context
const ClientVersion string = "clientversion"

// Version subcommand: Version shivas.
var Version = &subcommands.Command{
	UsageLine: "version",
	ShortDesc: "print shivas version",
	LongDesc:  "Print shivas version.",
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
	p, err := cipd.FindPackage("shivas", site.CipdInstalledPath)
	if err != nil {
		return err
	}
	ctx := context.Background()
	d, err := cipd.DescribePackage(ctx, p.Package, p.Pin.InstanceID)
	if err != nil {
		return err
	}

	fmt.Printf("shivas CLI tool: v%s+%s\n", VersionNumber, time.Time(d.RegisteredTs).Format("20060102150405"))
	fmt.Printf("CIPD Package:\t%s\n", p.Package)
	fmt.Printf("CIPD Version:\t%s\n", p.Pin.InstanceID)
	fmt.Printf("CIPD Updated:\t%s\n", d.RegisteredTs)
	fmt.Printf("CIPD Tracking:\t%s\n", p.Tracking)

	return nil
}
