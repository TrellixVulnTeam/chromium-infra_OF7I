// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"fmt"
	"sort"
)

// CommandWithFlags is a representation of a command with subcommands that takes a combination
// of flags, some of which have arguments and some of which don't. It also takes positional
// parameters, which are inserted after the subcommands and flags when serialized.
type CommandWithFlags struct {
	Commands       []string
	Flags          map[string][]string
	PositionalArgs []string
}

// ToCommand produces a list of arguments given a representation of a command with flags.
func (c *CommandWithFlags) ToCommand() []string {
	if c == nil {
		return nil
	}
	var out []string
	for _, s := range c.Commands {
		out = append(out, s)
	}
	// ToCommand must be deterministic, sort the keys before iterating.
	var keys []string
	for k := range c.Flags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		items := c.Flags[k]
		if len(items) == 0 {
			out = append(out, fmt.Sprintf("-%s", k))
			continue
		}
		for _, item := range items {
			out = append(out, fmt.Sprintf("-%s", k))
			out = append(out, item)
		}
	}
	for _, s := range c.PositionalArgs {
		out = append(out, s)
	}
	return out
}
