// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"fmt"
	"sort"
)

// An FlagEntry is a flag value. It can be nil, indicating a flag with no argument, or
// it can contain a string, which indicates a flag with a value associated with it.
type FlagEntry struct {
	Content string
}

// CommandWithFlags is a representation of a command with subcommands that takes a combination
// of flags, some of which have arguments and some of which don't. It also takes positional
// parameters, which are inserted after the subcommands and flags when serialized.
type CommandWithFlags struct {
	Commands       []string
	Flags          map[string]*FlagEntry
	PositionalArgs []string
}

// ToCommand produces a list of arguments given a representation of a command with flags.
func (c *CommandWithFlags) ToCommand() []string {
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
		v := c.Flags[k]
		out = append(out, fmt.Sprintf("-%s", k))
		if v != nil {
			out = append(out, v.Content)
		}
	}
	for _, s := range c.PositionalArgs {
		out = append(out, s)
	}
	return out
}

// ApplyFlagFilter takes a default decision (false for reject, true for keep) and a list of decisions
// for individual flags. It then applies the filter to either remove unknown flags or restrict the flags
// to a known subset of the flags.
func (c *CommandWithFlags) ApplyFlagFilter(keepByDefault bool, flags map[string]bool) *CommandWithFlags {
	for flag := range c.Flags {
		keep := keepByDefault
		if decision, ok := flags[flag]; ok {
			keep = decision
		}
		if !keep {
			delete(c.Flags, flag)
		}
	}
	return c
}
