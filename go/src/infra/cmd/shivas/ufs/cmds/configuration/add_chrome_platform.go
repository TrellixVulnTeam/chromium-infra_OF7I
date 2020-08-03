// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"fmt"
	"strings"

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

// AddChromePlatformCmd add ChromePlatform to the system.
var AddChromePlatformCmd = &subcommands.Command{
	UsageLine: "add-platform",
	ShortDesc: "Add platform configuration for browser machine",
	LongDesc:  cmdhelp.AddChromePlatformLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addChromePlatform{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.ChromePlatformFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.name, "name", "", "name of the platform")
		c.Flags.StringVar(&c.manufacturer, "manufacturer", "", "manufacturer name")
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags")
		c.Flags.StringVar(&c.description, "desc", "", "description for the platform")
		return c
	},
}

type addChromePlatform struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string
	interactive  bool

	name         string
	manufacturer string
	tags         string
	description  string
}

func (c *addChromePlatform) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addChromePlatform) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	var chromePlatform ufspb.ChromePlatform
	if c.interactive {
		utils.GetChromePlatformInteractiveInput(ctx, ic, &chromePlatform, false)
	} else {
		if c.newSpecsFile != "" {
			if err = utils.ParseJSONFile(c.newSpecsFile, &chromePlatform); err != nil {
				return err
			}
		} else {
			c.parseArgs(&chromePlatform)
		}
	}
	res, err := ic.CreateChromePlatform(ctx, &ufsAPI.CreateChromePlatformRequest{
		ChromePlatform:   &chromePlatform,
		ChromePlatformId: chromePlatform.GetName(),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	fmt.Printf("Successfully added the platform %s\n", res.Name)
	return nil
}

func (c *addChromePlatform) parseArgs(chromePlatform *ufspb.ChromePlatform) {
	chromePlatform.Name = c.name
	chromePlatform.Manufacturer = c.manufacturer
	chromePlatform.Tags = strings.Split(strings.Replace(c.tags, " ", "", -1), ",")
	chromePlatform.Description = c.description
}

func (c *addChromePlatform) validateArgs() error {
	if c.newSpecsFile != "" && c.interactive {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
	}
	if c.newSpecsFile != "" || c.interactive {
		if c.name != "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.manufacturer != "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-manufacturer' cannot be specified at the same time.")
		}
		if c.tags != "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-tags' cannot be specified at the same time.")
		}
		if c.description != "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-description' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.name == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so '-name' is required.")
		}
		if c.manufacturer == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so '-manufacturer' is required.")
		}
	}
	return nil
}
