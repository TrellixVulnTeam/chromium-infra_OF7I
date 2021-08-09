// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package meta

import (
	"context"
	"fmt"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/satlab/internal/site"
	"infra/libs/cipd"
)

// Version subcommand: the version of satlab
var Version = &subcommands.Command{
	UsageLine: "version",
	ShortDesc: "print satlab version",
	LongDesc:  "Print satlab version.",
	CommandRun: func() subcommands.CommandRun {
		c := &versionRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		return c
	},
}

// VersionRun store the command line arguments needed for the version subcommand.
type versionRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

// Run is the implementation of the satlab version subcommand.
func (c *versionRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// InnerRun writes the version to stdout and then exits.
func (c *versionRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	p, err := cipd.FindPackage("satlab", site.CipdInstalledPath)
	if err != nil {
		return errors.Annotate(err, "version").Err()
	}
	ctx := context.Background()
	d, err := cipd.DescribePackage(ctx, p.Package, p.Pin.InstanceID)
	if err != nil {
		return errors.Annotate(err, "version").Err()
	}

	fmt.Printf("satlab CLI tool: v%s+%s\n", site.VersionNumber, time.Time(d.RegisteredTs).Format("20060102150405"))
	fmt.Printf("CIPD Package:\t%s\n", p.Package)
	fmt.Printf("CIPD Version:\t%s\n", p.Pin.InstanceID)
	fmt.Printf("CIPD Updated:\t%s\n", d.RegisteredTs)
	fmt.Printf("CIPD Tracking:\t%s\n", p.Tracking)

	return nil
}
