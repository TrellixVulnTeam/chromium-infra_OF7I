// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"

	"github.com/maruel/subcommands"
)

var cmdModule = &subcommands.Command{
	UsageLine: "module [...]",
	ShortDesc: "deploys a single GAE module",
	LongDesc: `Deploys a single GAE module.

TODO: write more.
`,

	CommandRun: func() subcommands.CommandRun {
		c := &cmdModuleRun{}
		c.init()
		return c
	},
}

type cmdModuleRun struct {
	commandBase
}

func (c *cmdModuleRun) init() {
	c.commandBase.init(c.exec, extraFlags{
		appID:    true,
		tarball:  true,
		cacheDir: true,
		dryRun:   true,
	})
}

func (c *cmdModuleRun) exec(ctx context.Context) error {
	// TODO: implement.
	return nil
}
