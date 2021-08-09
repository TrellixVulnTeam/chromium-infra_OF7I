// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/flag"

	"infra/cmd/shivas/cmdhelp"
	"infra/cros/cmd/satlab/internal/site"
)

// MakeShivasFlags creates a map for the flags inherited from shivas.
func makeGetShivasFlags(c *getDUT) flagmap {
	out := make(flagmap)

	if len(c.zones) != 0 {
		out["zone"] = c.zones
	}
	if len(c.racks) != 0 {
		out["rack"] = c.racks
	}
	if len(c.machines) != 0 {
		out["machine"] = c.machines
	}
	if len(c.prototypes) != 0 {
		out["prototype"] = c.prototypes
	}
	if len(c.servos) != 0 {
		out["servo"] = c.servos
	}
	if len(c.servotypes) != 0 {
		out["servotype"] = c.servotypes
	}
	if len(c.switches) != 0 {
		out["switch"] = c.switches
	}
	if len(c.rpms) != 0 {
		out["rpms"] = c.rpms
	}
	if len(c.pools) != 0 {
		out["pools"] = c.pools
	}
	if c.wantHostInfoStore {
		out["host-info-store"] = []string{}
	}
	if c.outputFlags.JSON() {
		out["json"] = []string{}
	}
	return out
}

// ShivasGetDUT contains the arguments that can be used to get DUTs.
// It is inherited from shivas.
type shivasGetDUT struct {
	subcommands.CommandRunBase

	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	outputFlags site.OutputFlags
	commonFlags site.CommonFlags

	// Filters
	zones      []string
	racks      []string
	machines   []string
	prototypes []string
	tags       []string
	states     []string
	servos     []string
	servotypes []string
	switches   []string
	rpms       []string
	pools      []string

	pageSize          int
	keysOnly          bool
	wantHostInfoStore bool
}

// RegisterGetShivasFlags registers the flags inherited from shivas.
func registerGetShivasFlags(c *getDUT) {
	c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
	c.envFlags.Register(&c.Flags)
	c.outputFlags.Register(&c.Flags)
	c.commonFlags.Register(&c.Flags)

	c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
	c.Flags.BoolVar(&c.keysOnly, "keys", false, cmdhelp.KeysOnlyText)

	c.Flags.Var(flag.StringSlice(&c.zones), "zone", "Name(s) of a zone to filter by. Can be specified multiple times."+cmdhelp.ZoneFilterHelpText)
	c.Flags.Var(flag.StringSlice(&c.racks), "rack", "Name(s) of a rack to filter by. Can be specified multiple times.")
	c.Flags.Var(flag.StringSlice(&c.machines), "machine", "Name(s) of a machine/asset to filter by. Can be specified multiple times.")
	c.Flags.Var(flag.StringSlice(&c.prototypes), "prototype", "Name(s) of a host prototype to filter by. Can be specified multiple times.")
	c.Flags.Var(flag.StringSlice(&c.tags), "tag", "Name(s) of a tag to filter by. Can be specified multiple times.")
	c.Flags.Var(flag.StringSlice(&c.states), "state", "Name(s) of a state to filter by. Can be specified multiple times."+cmdhelp.StateFilterHelpText)
	c.Flags.Var(flag.StringSlice(&c.servos), "servo", "Name(s) of a servo:port to filter by. Can be specified multiple times.")
	c.Flags.Var(flag.StringSlice(&c.servotypes), "servotype", "Name(s) of a servo type to filter by. Can be specified multiple times.")
	c.Flags.Var(flag.StringSlice(&c.switches), "switch", "Name(s) of a switch to filter by. Can be specified multiple times.")
	c.Flags.Var(flag.StringSlice(&c.rpms), "rpm", "Name(s) of a rpm to filter by. Can be specified multiple times.")
	c.Flags.Var(flag.StringSlice(&c.pools), "pools", "Name(s) of a tag to filter by. Can be specified multiple times.")
	c.Flags.BoolVar(&c.wantHostInfoStore, "host-info-store", false, "write host info store to stdout")
}
