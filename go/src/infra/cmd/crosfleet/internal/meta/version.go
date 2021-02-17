// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package meta

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"

	"infra/cmd/crosfleet/internal/site"
	"infra/libs/cipd"
)

// Version subcommand: Version of crosfleet tool.
var Version = &subcommands.Command{
	UsageLine: "version",
	ShortDesc: "print crosfleet tool version",
	LongDesc:  "Print crosfleet tool version.",
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

// fallbackErrorMessage shows a fallback version message if the crosfleet tool is unable to
// locate its own CIPD package.
func fallbackErrorMessage(a subcommands.Application) {
	fmt.Fprintf(a.GetErr(), "Failed to find CIPD package!\n")
	fmt.Printf("crosfleet CLI Tool: v%d\n", site.VersionNumber)
}

func (c *versionRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	var err error

	p, err := findCrosfleetPackage()
	if err != nil {
		fallbackErrorMessage(a)
		return err
	}
	ctx := context.Background()
	d, err := describe(ctx, p.Package, p.Pin.InstanceID)
	if err != nil {
		fallbackErrorMessage(a)
		return err
	}

	fmt.Printf(fmt.Sprintf("crosfleet CLI tool: v%d+%s\n", site.VersionNumber, time.Time(d.RegisteredTs).Format("20060102150405")))
	fmt.Printf("CIPD Package:\t%s\n", p.Package)
	fmt.Printf("CIPD Version:\t%s\n", p.Pin.InstanceID)
	fmt.Printf("CIPD Updated:\t%s\n", d.RegisteredTs)
	fmt.Printf("CIPD Tracking:\t%s\n", p.Tracking)
	return nil
}

func findCrosfleetPackage() (*cipd.Package, error) {
	d, err := executableDir()
	if err != nil {
		return nil, errors.Annotate(err, "find crosfleet package").Err()
	}
	root, err := findCIPDRootDir(d)
	if err != nil {
		return nil, errors.Annotate(err, "find crosfleet package").Err()
	}
	pkgs, err := cipd.InstalledPackages("crosfleet")(root)
	if err != nil {
		return nil, errors.Annotate(err, "find crosfleet package").Err()
	}
	for _, p := range pkgs {
		if !strings.HasPrefix(p.Package, "chromiumos/infra/crosfleet/") {
			continue
		}
		return &p, nil
	}
	return nil, errors.Reason("find crosfleet package: not found").Err()
}
