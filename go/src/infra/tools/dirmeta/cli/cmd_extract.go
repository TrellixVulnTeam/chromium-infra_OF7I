// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"github.com/maruel/subcommands"
)

func cmdExtract() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `extract`,
		ShortDesc: "extract metadata from a directory tree",
		LongDesc:  "Extract metadata from a directory tree",
		CommandRun: func() subcommands.CommandRun {
			r := &extractRun{}
			return r
		},
	}
}

type extractRun struct {
	baseCommandRun
}

func (r *extractRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	panic("not implemented")
}
