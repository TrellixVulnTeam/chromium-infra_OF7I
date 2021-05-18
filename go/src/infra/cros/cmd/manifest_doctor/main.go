// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	luciflag "go.chromium.org/luci/common/flag"
)

func cmdLocalManifestBrancher() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "branch-local-manifest --chromeos_checkout ~/chromiumos " +
			" --min_milestone 90 --projects chromeos/project/foo,chromeos/project/bar",
		ShortDesc: "Repair local_manifest.xml on specified non-ToT branches.",
		CommandRun: func() subcommands.CommandRun {
			b := &localManifestBrancher{}
			b.Flags.StringVar(&b.chromeosCheckoutPath, "chromeos_checkout", "",
				"Path to full ChromeOS checkout.")
			b.Flags.IntVar(&b.minMilestone, "min_milestone", -1,
				"Minimum milestone of branches to consider. Used directly "+
					"in selecting release branches and indirectly for others.")
			b.Flags.Var(luciflag.CommaList(&b.projects), "projects",
				"Comma-separated list of project paths to consider. "+
					"At least one project is required.")
			return b
		}}
}

func (b *localManifestBrancher) validate() error {
	if b.minMilestone == -1 {
		return fmt.Errorf("--min_milestone required")
	}

	if b.chromeosCheckoutPath == "" {
		return fmt.Errorf("--chromeos_checkout required")
	} else if _, err := os.Stat(b.chromeosCheckoutPath); os.IsNotExist(err) {
		return fmt.Errorf("path %s does not exist", b.chromeosCheckoutPath)
	}

	if len(b.projects) == 0 {
		return fmt.Errorf("at least one project is required")
	}

	return nil
}

func (b *localManifestBrancher) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	flag.Parse()

	if err := b.validate(); err != nil {
		log.Printf("error validating args: %v", err)
		return 1
	}

	if err := BranchLocalManifests(b.chromeosCheckoutPath, b.projects, b.minMilestone); err != nil {
		log.Printf(err.Error())
		return 2
	}

	return 0
}

type localManifestBrancher struct {
	subcommands.CommandRunBase
	chromeosCheckoutPath string
	minMilestone         int
	projectList          string
	projects             []string
}

// GetApplication returns an instance of the application.
func GetApplication() *cli.Application {
	return &cli.Application{
		Name: "manifest_doctor",

		Context: func(ctx context.Context) context.Context {
			return ctx
		},

		Commands: []*subcommands.Command{
			cmdLocalManifestBrancher(),
		},
	}
}

func main() {
	app := GetApplication()
	os.Exit(subcommands.Run(app, nil))
}
