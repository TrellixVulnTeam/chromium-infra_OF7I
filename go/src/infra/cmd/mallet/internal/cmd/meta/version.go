// Copyright 2020 The Chromium Authors. All rights reserved.
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

	"infra/cmd/mallet/internal/site"
	"infra/libs/cipd"
)

// Version subcommand: Version skylab tool.
var Version = &subcommands.Command{
	UsageLine: "version",
	ShortDesc: "print mallet tool version",
	LongDesc:  "Print mallet tool version.",
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

// fallbackErrorMessage shows a fallback version message if the mallet tool is unable to
// locate its own CIPD package.
func fallbackErrorMessage(a subcommands.Application) {
	fmt.Fprintf(a.GetErr(), "Failed to find CIPD package!\n")
	fmt.Printf("mallet CLI Tool: v%s\n", site.VersionNumber)
}

func (c *versionRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	var err error

	p, err := findMalletPackage()
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

	fmt.Printf(fmt.Sprintf("mallet CLI tool: v%s+%s\n", site.VersionNumber, time.Time(d.RegisteredTs).Format("20060102150405")))
	fmt.Printf("CIPD Package:\t%s\n", p.Package)
	fmt.Printf("CIPD Version:\t%s\n", p.Pin.InstanceID)
	fmt.Printf("CIPD Updated:\t%s\n", d.RegisteredTs)
	fmt.Printf("CIPD Tracking:\t%s\n", p.Tracking)
	return nil
}

func findMalletPackage() (*cipd.Package, error) {
	d, err := executableDir()
	if err != nil {
		return nil, errors.Annotate(err, "find mallet package").Err()
	}
	root, err := findCIPDRootDir(d)
	if err != nil {
		return nil, errors.Annotate(err, "find mallet package").Err()
	}
	pkgs, err := cipd.InstalledPackages("mallet")(root)
	if err != nil {
		return nil, errors.Annotate(err, "find mallet package").Err()
	}
	for _, p := range pkgs {
		if !strings.HasPrefix(p.Package, "chromiumos/infra/mallet/") {
			continue
		}
		return &p, nil
	}
	return nil, errors.Reason("find mallet package: not found").Err()
}
