// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package switches

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// AddSwitchCmd add Switch in the lab.
var AddSwitchCmd = &subcommands.Command{
	UsageLine: "add-switch [Options...]",
	ShortDesc: "Add a switch to UFS",
	LongDesc:  cmdhelp.AddSwitchLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addSwitch{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.SwitchFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.rackName, "rack", "", "name of the rack to associate the switch")
		c.Flags.StringVar(&c.switchName, "name", "", "the name of the switch to add")
		c.Flags.StringVar(&c.description, "description", "", "the description of the switch to add")
		c.Flags.IntVar(&c.capacity, "capacity", 0, "indicate how many ports this switch support")
		return c
	},
}

type addSwitch struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string
	interactive  bool

	rackName    string
	switchName  string
	description string
	capacity    int
}

func (c *addSwitch) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addSwitch) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	if c.commonFlags.Verbose() {
		fmt.Printf("Using UFS service %s\n", e.UnifiedFleetService)
	}
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})

	var s ufspb.Switch
	if c.interactive {
		c.rackName = utils.GetSwitchInteractiveInput(ctx, ic, &s, false)
	} else {
		if c.newSpecsFile != "" {
			if err = utils.ParseJSONFile(c.newSpecsFile, &s); err != nil {
				return err
			}
		} else {
			c.parseArgs(&s)
		}
	}
	res, err := ic.CreateSwitch(ctx, &ufsAPI.CreateSwitchRequest{
		Switch:   &s,
		SwitchId: s.GetName(),
		Rack:     c.rackName,
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	fmt.Printf("Successfully added the switch %s to rack %s\n", res.Name, c.rackName)
	return nil
}

func (c *addSwitch) parseArgs(s *ufspb.Switch) {
	s.Name = c.switchName
	s.Description = c.description
	s.CapacityPort = int32(c.capacity)
}

func (c *addSwitch) validateArgs() error {
	if c.newSpecsFile != "" || c.interactive {
		if c.switchName != "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.capacity != 0 {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-capacity' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile != "" {
		if c.interactive {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
		}
		if c.rackName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nRack name (-rack) is required for JSON mode.")
		}
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.switchName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so '-name' is required.")
		}
		if c.capacity == 0 {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so '-capacity' is required.")
		}
		if c.rackName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nRack name (-rack) is required.")
		}
	}
	return nil
}
