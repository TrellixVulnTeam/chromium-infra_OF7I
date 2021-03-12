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

	"infra/cros/internal/chromeosversion"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
)

func cmdBumpVersion() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "bump-version",
		ShortDesc: "Bump the specified component in chromeos_version.sh.",
		CommandRun: func() subcommands.CommandRun {
			b := &bumpVersion{}
			b.Flags.StringVar(&b.chromiumosOverlayRepo, "chromiumos_overlay_repo", "",
				"Full path to the local checkout of the chromiumos-overlay project.")
			b.Flags.BoolVar(&b.bumpMilestoneComponent, "bump_milestone_component", false,
				"Bump the milestone version component (e.g. 89 in R89.13729.0.0).")
			b.Flags.BoolVar(&b.bumpBuildComponent, "bump_build_component", false,
				"Bump the build version component (e.g. 13729 in R89.13729.0.0).")
			b.Flags.BoolVar(&b.bumpBranchComponent, "bump_branch_component", false,
				"Bump the branch version component (e.g. 45 in R89.13729.45.0).")
			b.Flags.BoolVar(&b.bumpPatchComponent, "bump_patch_component", false,
				"Bump the milestone version component (e.g. 1 in R89.13729.45.1).")
			b.Flags.StringVar(&b.bumpFromBranchName, "bump_from_branch_name", "",
				"Infer which component to bump from the branch name "+
					"(e.g. firmware-atlas-11827.B).")
			return b
		}}
}

func (b *bumpVersion) componentToBump() (chromeosversion.VersionComponent, error) {
	flags := []bool{
		b.bumpMilestoneComponent,
		b.bumpBuildComponent,
		b.bumpBranchComponent,
		b.bumpPatchComponent,
		b.bumpFromBranchName != "",
	}
	count := 0
	for _, t := range flags {
		if t {
			count++
		}
	}

	if count > 1 {
		return chromeosversion.Unspecified,
			fmt.Errorf("only one of --bump_milestone_component, --bump_build_component, " +
				"--bump_branch_component, --bump_patch_component, and " +
				"--bump_from_branch_name can be set")
	} else if b.bumpMilestoneComponent {
		return chromeosversion.ChromeBranch, nil
	} else if b.bumpBuildComponent {
		return chromeosversion.Build, nil
	} else if b.bumpBranchComponent {
		return chromeosversion.Branch, nil
	} else if b.bumpPatchComponent {
		return chromeosversion.Patch, nil
	} else if branch := b.bumpFromBranchName; branch != "" {
		return chromeosversion.ComponentToBumpFromBranch(branch), nil
	}
	return chromeosversion.Unspecified, fmt.Errorf("no component specified")
}

func (b *bumpVersion) validate() error {
	if b.chromiumosOverlayRepo == "" {
		return fmt.Errorf("--chromiumos_overlay_repo required")
	} else if _, err := os.Stat(b.chromiumosOverlayRepo); os.IsNotExist(err) {
		return fmt.Errorf("path %s does not exist", b.chromiumosOverlayRepo)
	}

	_, err := b.componentToBump()
	if err != nil {
		return err
	}

	return nil
}

func (b *bumpVersion) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	flag.Parse()

	if err := b.validate(); err != nil {
		log.Printf("error validating args: %v", err)
		return 1
	}

	vinfo, err := chromeosversion.GetVersionInfoFromRepo(b.chromiumosOverlayRepo)
	if err != nil {
		log.Printf("error getting version info from repo: %v", err)
		return 2
	}

	componentToBump, _ := b.componentToBump()
	vinfo.IncrementVersion(componentToBump)

	if err := vinfo.UpdateVersionFile(); err != nil {
		log.Printf("error updating version file: %v", err)
		return 3
	}

	return 0
}

type bumpVersion struct {
	subcommands.CommandRunBase
	chromiumosOverlayRepo  string
	bumpMilestoneComponent bool
	bumpBuildComponent     bool
	bumpBranchComponent    bool
	bumpPatchComponent     bool
	bumpFromBranchName     string
}

// GetApplication returns an instance of the application.
func GetApplication() *cli.Application {
	return &cli.Application{
		Name: "version_bumper",

		Context: func(ctx context.Context) context.Context {
			return ctx
		},

		Commands: []*subcommands.Command{
			cmdBumpVersion(),
		},
	}
}

func main() {
	app := GetApplication()
	os.Exit(subcommands.Run(app, nil))
}
