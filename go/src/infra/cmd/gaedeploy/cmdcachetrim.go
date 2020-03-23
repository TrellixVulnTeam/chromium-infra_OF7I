// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"

	"github.com/maruel/subcommands"
)

var cmdCacheTrim = &subcommands.Command{
	Advanced:  true,
	UsageLine: "cache-trim [...]",
	ShortDesc: "trims the local cache of unpacked tarballs",
	LongDesc: `Trims the local cache of unpacked tarballs.

Sorts cache entries by last access time and removes oldest ones until only
the amount specified by -keep remain.
`,

	CommandRun: func() subcommands.CommandRun {
		c := &cmdCacheTrimRun{}
		c.init()
		return c
	},
}

type cmdCacheTrimRun struct {
	commandBase

	keep int // -keep flag
}

func (c *cmdCacheTrimRun) init() {
	c.commandBase.init(c.exec, extraFlags{
		cacheDir: true,
	})
	c.Flags.IntVar(&c.keep, "keep", 50, "How many cache entries to keep (default is 50).")
}

func (c *cmdCacheTrimRun) exec(ctx context.Context) error {
	if c.keep < 0 {
		return errBadFlag("-keep", "must be non-negative")
	}
	return c.cache.Trim(ctx, c.keep)
}
