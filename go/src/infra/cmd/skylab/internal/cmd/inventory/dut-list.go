// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	flagx "infra/cmd/skylab/internal/flagx"
	inv "infra/cmd/skylab/internal/inventory"
	"infra/cmd/skylab/internal/site"
	"infra/cmdsupport/cmdlib"
	"infra/libs/skylab/swarming"
)

// DutList subcommand: Get DUT information
var DutList = &subcommands.Command{
	UsageLine: "dut-list [-pool POOL] [-model MODEL] [-board BOARD]",
	ShortDesc: "List hostnames of devices matching search criteria",
	LongDesc: `List hostnames of devices matching search criteria.

	If no criteria are provided, dut-list will list all the DUT hostnames in Skylab.

	Search criteria include pool, model, board`,
	CommandRun: func() subcommands.CommandRun {
		c := &dutListRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.board, "board", "", "board name")
		c.Flags.StringVar(&c.model, "model", "", "model name")
		c.Flags.StringVar(&c.pool, "pool", "", "pool name")
		c.Flags.StringVar(&c.servoType, "servo-type", "", "the type of servo")
		c.Flags.BoolVar(&c.useInventory, "use-inventory", false, "use the inventory service if set, use swarming if unset")
		c.Flags.Var(flagx.Dims(&c.dims), "dims", "List of additional dimensions in format key1=value1,key2=value2,... .")
		return c
	},
}

type dutListRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  skycmdlib.EnvFlags

	board        string
	model        string
	pool         string
	servoType    string
	useInventory bool
	// nil is a valid value for dims.
	dims map[string]string
}

func (c *dutListRun) getUseInventory() bool {
	return c.useInventory
}

func (c *dutListRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, errors.Annotate(err, "dut-list").Err())
		return 1
	}
	return 0
}

func (c *dutListRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if len(args) != 0 {
		return cmdlib.NewUsageError(c.Flags, "unexpected positional argument.")
	}

	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}

	params := dutListParams{
		board:     c.board,
		model:     c.model,
		pool:      c.pool,
		servoType: c.servoType,
		dims:      c.dims,
	}
	var listed []*swarming.ListedHost

	if c.getUseInventory() {
		// TODO(gregorynisbet): remove this
		panic("QUERYING INVENTORY DIRECTLY NOT YET SUPPORTED")
	} else {
		sc, err := swarming.NewClient(hc, c.envFlags.Env().SwarmingService)
		if err != nil {
			return err
		}
		listed, err = listDutsSwarming(ctx, sc, params)
		if err != nil {
			return err
		}
	}

	ic := inv.NewInventoryClient(hc, c.envFlags.Env())

	var rawSwarmingHosts []string
	for _, l := range listed {
		rawSwarmingHosts = append(rawSwarmingHosts, l.String())
	}
	hosts, err := ic.FilterDUTHostnames(ctx, rawSwarmingHosts)
	if err != nil {
		return err
	}

	for _, l := range hosts {
		fmt.Printf("%s\n", l)
	}

	return nil
}

type dutListParams struct {
	board     string
	model     string
	pool      string
	servoType string
	// dims is the additional dimensions used to filter DUTs.
	// nil is a valid value for dims.
	dims map[string]string
}

func (p dutListParams) swarmingDims() ([]*swarming_api.SwarmingRpcsStringPair, error) {
	// TODO(gregorynisbet): support servoType
	if p.servoType != "" {
		return nil, errors.New("servoType not yet supported")
	}
	makePair := func(key string, value string) *swarming_api.SwarmingRpcsStringPair {
		out := &swarming_api.SwarmingRpcsStringPair{}
		out.Key = key
		out.Value = value
		return out
	}
	out := []*swarming_api.SwarmingRpcsStringPair{makePair("pool", "ChromeOSSkylab")}
	if p.model != "" {
		out = append(out, makePair("label-model", p.model))
	}
	if p.board != "" {
		out = append(out, makePair("label-board", p.board))
	}
	if p.pool != "" {
		out = append(out, makePair("label-pool", p.pool))
	}
	for k, v := range p.dims {
		out = append(out, makePair(k, v))
	}
	return out, nil
}

func listDutsSwarming(ctx context.Context, sc *swarming.Client, params dutListParams) ([]*swarming.ListedHost, error) {
	dims, err := params.swarmingDims()
	if err != nil {
		return nil, err
	}
	listed, err := sc.GetListedBots(ctx, dims)
	if err != nil {
		return nil, err
	}
	return listed, nil
}
