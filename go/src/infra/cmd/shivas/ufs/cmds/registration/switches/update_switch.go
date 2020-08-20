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

// UpdateSwitchCmd Update switch by given name.
var UpdateSwitchCmd = &subcommands.Command{
	UsageLine: "update-switch [Options...]",
	ShortDesc: "Update a switch by name",
	LongDesc:  cmdhelp.UpdateSwitchLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateSwitch{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.SwitchFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.rackName, "rack", "", "name of the rack to associate the switch")
		c.Flags.StringVar(&c.switchName, "name", "", "the name of the switch to update")
		c.Flags.StringVar(&c.description, "desc", "", "the description of the switch to update ."+cmdhelp.ClearFieldHelpText)
		c.Flags.IntVar(&c.capacity, "capacity", 0, "indicate how many ports this switch support. "+"To clear this field set it to -1.")
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags. You can only append/add new tags here. "+cmdhelp.ClearFieldHelpText)
		return c
	},
}

type updateSwitch struct {
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
	tags        string
}

func (c *updateSwitch) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateSwitch) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
		c.rackName = utils.GetSwitchInteractiveInput(ctx, ic, &s, true)
	} else {
		if c.newSpecsFile != "" {
			if err = utils.ParseJSONFile(c.newSpecsFile, &s); err != nil {
				return err
			}
		} else {
			c.parseArgs(&s)
		}
	}
	if err := utils.PrintExistingSwitch(ctx, ic, s.Name); err != nil {
		return err
	}
	s.Name = ufsUtil.AddPrefix(ufsUtil.SwitchCollection, s.Name)
	res, err := ic.UpdateSwitch(ctx, &ufsAPI.UpdateSwitchRequest{
		Switch: &s,
		Rack:   c.rackName,
		UpdateMask: utils.GetUpdateMask(&c.Flags, map[string]string{
			"rack":     "rack",
			"capacity": "capacity",
			"tags":     "tags",
			"desc":     "description",
		}),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The switch after update:")
	utils.PrintProtoJSON(res, false)
	fmt.Printf("Successfully updated the switch %s\n", res.Name)
	return nil
}

func (c *updateSwitch) parseArgs(s *ufspb.Switch) {
	s.Name = c.switchName
	if c.description == utils.ClearFieldValue {
		s.Description = ""
	} else {
		s.Description = c.description
	}
	if c.capacity == -1 {
		s.CapacityPort = 0
	} else {
		s.CapacityPort = int32(c.capacity)
	}
	if c.tags == utils.ClearFieldValue {
		s.Tags = nil
	} else {
		s.Tags = utils.GetStringSlice(c.tags)
	}
}

func (c *updateSwitch) validateArgs() error {
	if c.newSpecsFile != "" && c.interactive {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
	}
	if c.newSpecsFile != "" || c.interactive {
		if c.switchName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.capacity != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-capacity' cannot be specified at the same time.")
		}
		if c.description != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-desc' cannot be specified at the same time.")
		}
		if c.tags != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-tags' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.switchName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f' or '-i') is specified.")
		}
		if c.rackName == "" && c.capacity == 0 && c.description == "" && c.tags == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNothing to update. Please provide any field to update")
		}
	}
	return nil
}
