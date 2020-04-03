// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/cmd/gaedeploy/gcloud"
)

var cmdCleanup = &subcommands.Command{
	UsageLine: "cleanup [...]",
	ShortDesc: "deletes oldest GAE versions with no traffic assignment",
	LongDesc: `Deletes oldest GAE versions with no traffic assignment.

Lists all deployed versions of all modules and deletes inactive ones, i.e. ones
that are assigned no traffic in the traffic splitting configuration.

Keeps all versions that receive traffic (regardless of their age), and also up
to -keep of most recently deployed inactive versions.
`,

	CommandRun: func() subcommands.CommandRun {
		c := &cmdCleanupRun{}
		c.init()
		return c
	},
}

type cmdCleanupRun struct {
	commandBase

	keep int // -keep flag
}

func (c *cmdCleanupRun) init() {
	c.commandBase.init(c.exec, extraFlags{
		appID:  true,
		dryRun: true,
	})
	c.Flags.IntVar(&c.keep, "keep", 5, "How many inactive versions to keep (default is 5).")
}

func (c *cmdCleanupRun) exec(ctx context.Context) error {
	if c.keep < 0 {
		return errBadFlag("-keep", "must be non-negative")
	}

	logging.Infof(ctx, "App ID: %s", c.appID)
	logging.Infof(ctx, "Keep:   %d", c.keep)

	mods, err := gcloud.List(ctx, c.appID, "")
	if err != nil {
		return errors.Annotate(err, "failed to app versions").Err()
	}

	// Visit modules in deterministic order, cleanup each one separately.
	names := make([]string, 0, len(mods))
	for mod := range mods {
		names = append(names, mod)
	}
	sort.Strings(names)
	for _, mod := range names {
		if err := cleanupVersions(ctx, c.appID, mod, mods[mod], c.keep, c.dryRun); err != nil {
			return errors.Annotate(err, "failed to cleanup module %q", mod).Err()
		}
	}
	return nil
}

// cleanupVersions cleans up old inactive version of some single module.
func cleanupVersions(ctx context.Context, appID, module string, vers gcloud.Versions, keepNum int, dryRun bool) error {
	logging.Infof(ctx, "Module %q:", module)

	list := make([]gcloud.Version, 0, len(vers))
	for _, ver := range vers {
		list = append(list, ver)
	}

	// Most recent first.
	sort.Slice(list, func(i, j int) bool {
		return list[i].LastDeployedTime.After(list[j].LastDeployedTime)
	})

	// Log all versions along with the verdict for them, to get the whole picture.
	var versionsToDelete []string
	for _, v := range list {
		verdict := ""
		if v.TrafficSplit == 0.0 {
			if keepNum == 0 {
				versionsToDelete = append(versionsToDelete, v.Name)
				verdict = "deleting: too old"
			} else {
				keepNum--
				verdict = "keeping: recent enough"
			}
		} else {
			verdict = "keeping: receives traffic"
		}

		padding := 16 - len(v.Name)
		if padding < 0 {
			padding = 0
		}

		logging.Infof(ctx, "  %s %s %.1f traffic, deployed %s, %s",
			v.Name, strings.Repeat(" ", padding),
			v.TrafficSplit, humanize.Time(v.LastDeployedTime), verdict)
	}

	if len(versionsToDelete) == 0 {
		logging.Infof(ctx, "Nothing to cleanup in %s.", module)
		return nil
	}

	logging.Infof(ctx, "Deleting %s...", strings.Join(versionsToDelete, ", "))
	return gcloud.Run(ctx, append([]string{
		"app", "versions", "delete",
		"--quiet", // disable interactive prompts
		"--project", appID,
		"--service", module,
	}, versionsToDelete...), "", nil, dryRun)
}
