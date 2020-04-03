// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/flag/stringlistflag"
	"go.chromium.org/luci/common/logging"

	"infra/cmd/gaedeploy/gcloud"
)

var cmdYaml = &subcommands.Command{
	UsageLine: "yaml [...]",
	ShortDesc: "deploys one or more deployable GAE YAML (e.g. dispatch.yaml)",
	LongDesc: `Deploys one or more deployable GAE YAML (e.g. dispatch.yaml).

Fetches and unpacks the tarball, then simply calls:
	gcloud app deploy --project <app-id> [list of deployable yamls]

Where paths to deployable YAMLs are provided via repeated "-deployable-yaml"
flag. Only specified YAMLs will be deployed.
`,

	CommandRun: func() subcommands.CommandRun {
		c := &cmdYamlRun{}
		c.init()
		return c
	},
}

type cmdYamlRun struct {
	commandBase

	deployableYaml stringlistflag.Flag // -deployable-yaml flag, required
}

func (c *cmdYamlRun) init() {
	c.commandBase.init(c.exec, extraFlags{
		appID:    true,
		tarball:  true,
		cacheDir: true,
		dryRun:   true,
	})

	c.Flags.Var(&c.deployableYaml, "deployable-yaml", "Path within the tarball to a YAML to deploy (can be repeated multiple times).")
}

func (c *cmdYamlRun) exec(ctx context.Context) error {
	if len(c.deployableYaml) == 0 {
		return errBadFlag("-deployable-yaml", "a value is required")
	}

	// Scope this command only to non-module YAMLs. There's a separate command
	// for modules. They are significantly different.
	allowedYAMLs := stringset.NewFromSlice(
		"cron.yaml",
		"dispatch.yaml",
		"dos.yaml",
		"index.yaml",
		"queue.yaml",
	)
	for _, p := range c.deployableYaml {
		if !allowedYAMLs.Has(filepath.Base(p)) {
			return errBadFlag("-deployable-yaml", fmt.Sprintf("%s is not a valid target", p))
		}
	}

	logging.Infof(ctx, "App ID:  %s", c.appID)
	logging.Infof(ctx, "Tarball: %s", c.tarballSource)
	logging.Infof(ctx, "Cache:   %s", c.cache.Root)
	logging.Infof(ctx, "YAML(s): %s", c.deployableYaml)

	return c.cache.WithTarball(ctx, c.source, func(path string) error {
		for _, localPath := range c.deployableYaml {
			if _, err := os.Stat(filepath.Join(path, localPath)); err != nil {
				return errors.Annotate(err, "bad YAML file %q", localPath).Err()
			}
		}
		return gcloud.Run(ctx, append([]string{
			"app", "deploy",
			"--project", c.appID,
			"--quiet", // disable interactive prompts
		}, c.deployableYaml...), path, nil, c.dryRun)
	})
}
