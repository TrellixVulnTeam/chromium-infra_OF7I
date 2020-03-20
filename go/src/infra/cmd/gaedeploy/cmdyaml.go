// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"

	"github.com/maruel/subcommands"
)

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
}

func (c *cmdYamlRun) init() {
	c.commandBase.init(c.exec)
}

func (c *cmdYamlRun) exec(ctx context.Context) error {
	// TODO: implement.
	return nil
}
