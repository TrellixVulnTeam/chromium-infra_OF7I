// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/logging"
)

const deployableYamlPlaceholder = "<path>"

var cmdYaml = &subcommands.Command{
	UsageLine: "yaml [...]",
	ShortDesc: "deploys a single deployable GAE YAML (e.g. dispatch.yaml)",
	LongDesc: `Deploys a single deployable GAE YAML (e.g. dispatch.yaml).

TODO: write more.
`,

	CommandRun: func() subcommands.CommandRun {
		c := &cmdYamlRun{}
		c.init()
		return c
	},
}

type cmdYamlRun struct {
	commandBase

	deployableYaml string // -deployable-yaml flag, required
}

func (c *cmdYamlRun) init() {
	c.commandBase.init(c.exec)

	c.Flags.StringVar(&c.deployableYaml, "deployable-yaml", deployableYamlPlaceholder,
		"Path within the tarball to a YAML to deploy.")
}

func (c *cmdYamlRun) exec(ctx context.Context) error {
	if c.deployableYaml == deployableYamlPlaceholder {
		return errBadFlag("-deployable-yaml", "a value is required")
	}

	logging.Infof(ctx, "App ID:  %s", c.appID)
	logging.Infof(ctx, "Tarball: %s", c.tarballSource)
	logging.Infof(ctx, "Cache:   %s", c.cacheDir)
	logging.Infof(ctx, "YAML:    %s", c.deployableYaml)

	return c.cache.WithTarball(ctx, c.source, func(path string) error {
		// TODO: implement.
		return nil
	})
}
