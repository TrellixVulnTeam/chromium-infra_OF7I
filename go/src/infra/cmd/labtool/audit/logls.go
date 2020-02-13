// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package audit

import (
	"sort"

	"github.com/maruel/subcommands"

	"infra/cmd/labtool/utils"
	"infra/cmdsupport/cmdlib"
)

// LoglsCmd prints previous audit runs' info.
var LoglsCmd = &subcommands.Command{
	UsageLine: "ls-logs",
	ShortDesc: "List log of previous audits on this machine",
	LongDesc:  "Lists the details of the previous audit runs on the machine",
	CommandRun: func() subcommands.CommandRun {
		c := &logls{}
		// Attempt to use home dir for logs, failing which use /tmp
		c.Flags.StringVar(&c.logDir, "log-dir", getLogDir(), "Dir to store logs for any runs of scan.")
		c.Flags.IntVar(&c.limit, "limit", 20, "Limit to latest")
		c.Flags.IntVar(&c.index, "index", -1, `Index of the log. When set the statistics
of that particular run are returned, including infos about successful and failed assets.`)
		return c
	},
}

type logls struct {
	subcommands.CommandRunBase
	logDir string
	limit  int
	index  int
}

func (c *logls) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *logls) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	logs, err := utils.ListLogs(c.logDir)
	if err != nil {
		return err
	}
	sort.Sort(logs)
	if c.index >= 0 && c.index < len(logs) {
		utils.PrintLogStatsAndResult(logs[c.index], c.index)
	} else {
		utils.PrintLogStats(logs, c.limit)
	}
	return nil
}
