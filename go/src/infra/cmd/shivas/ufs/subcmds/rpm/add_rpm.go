// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpm

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// AddRPMCmd add RPM in the lab.
var AddRPMCmd = &subcommands.Command{
	UsageLine: "rpm [Options...]",
	ShortDesc: "Add a rpm to a rack",
	LongDesc:  cmdhelp.AddRPMLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addRPM{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.RPMFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.rackName, "rack", "", "name of the rack to associate the rpm")
		c.Flags.StringVar(&c.rpmName, "name", "", "the name of the rpm to add")
		c.Flags.StringVar(&c.macAddress, "mac", "", "the mac address of the rpm to add")
		c.Flags.StringVar(&c.description, "desc", "", "the description of the rpm to add")
		c.Flags.IntVar(&c.capacity, "capacity", 0, "indicate how many ports this rpm support")
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags. You can only append/add new tags here.")
		return c
	},
}

type addRPM struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string
	interactive  bool

	rackName    string
	rpmName     string
	macAddress  string
	description string
	capacity    int
	tags        string
}

func (c *addRPM) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addRPM) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	ns, err := c.envFlags.Namespace()
	if err != nil {
		return err
	}
	ctx = utils.SetupContext(ctx, ns)
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

	var rpm ufspb.RPM
	if c.interactive {
		return errors.New("Interactive mode for this " +
			"command is not yet implemented yet.")
		//utils.GetRPMInteractiveInput(ctx, ic, &rpm, false)
	} else if c.newSpecsFile != "" {
		if err = utils.ParseJSONFile(c.newSpecsFile, &rpm); err != nil {
			return err
		}
		if rpm.GetRack() == "" {
			return errors.New(fmt.Sprintf("rack field is empty in json. It is a required parameter for json input."))
		}
	} else {
		c.parseArgs(&rpm)
	}
	res, err := ic.CreateRPM(ctx, &ufsAPI.CreateRPMRequest{
		RPM:   &rpm,
		RPMId: rpm.GetName(),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Printf("Successfully added the rpm %s to rack %s\n", res.Name, res.GetRack())
	return nil
}

func (c *addRPM) parseArgs(rpm *ufspb.RPM) {
	rpm.Name = c.rpmName
	rpm.MacAddress = c.macAddress
	rpm.Tags = utils.GetStringSlice(c.tags)
	rpm.Description = c.description
	rpm.CapacityPort = int32(c.capacity)
	rpm.Rack = c.rackName
}

func (c *addRPM) validateArgs() error {
	if c.newSpecsFile != "" && c.interactive {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
	}
	if c.newSpecsFile != "" || c.interactive {
		if c.rpmName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.macAddress != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-mac' cannot be specified at the same time.")
		}
		if c.tags != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-tags' cannot be specified at the same time.")
		}
		if c.rackName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-rack' cannot be specified at the same time.")
		}
		if c.capacity != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-capacity' cannot be specified at the same time.")
		}
		if c.description != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-desc' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.rpmName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f' or '-i') is specified.")
		}
		if c.macAddress == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-mac' is required, no mode ('-f' or '-i') is specified.")
		}
		if c.capacity == 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-capacity' is required, no mode ('-f' or '-i') is specified.")
		}
		if c.rackName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nRack name (-rack) is required.")
		}
	}
	return nil
}
