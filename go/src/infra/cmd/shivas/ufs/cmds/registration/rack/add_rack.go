// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rack

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

// AddRackCmd add Rack to the system.
var AddRackCmd = &subcommands.Command{
	UsageLine: "add-rack [Options...]",
	ShortDesc: "Add a rack to UFS",
	LongDesc:  cmdhelp.AddRackLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addRack{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.RackRegistrationFileText)

		c.Flags.StringVar(&c.rackName, "name", "", "the name of the rack to add")
		c.Flags.StringVar(&c.labName, "lab", "", fmt.Sprintf("the name of the lab to add the rack to. Valid lab strings: [%s]", strings.Join(utils.ValidLabStr(), ", ")))
		c.Flags.IntVar(&c.capacity, "capacity", 0, "indicate how many machines can be added to this rack")
		return c
	},
}

type addRack struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     site.EnvFlags
	commonFlags  site.CommonFlags
	newSpecsFile string
	rackName     string
	labName      string
	capacity     int
}

func (c *addRack) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addRack) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	var rackRegistrationReq ufsAPI.RackRegistrationRequest
	if c.newSpecsFile != "" {
		if err := utils.ParseJSONFile(c.newSpecsFile, &rackRegistrationReq); err != nil {
			return err
		}
		ufsLab := utils.ToUFSLab(c.labName)
		if rackRegistrationReq.GetRack().Location == nil {
			rackRegistrationReq.GetRack().Location = &ufspb.Location{}
		}
		rackRegistrationReq.GetRack().GetLocation().Lab = ufsLab
		rackRegistrationReq.GetRack().Realm = utils.ToUFSRealm(ufsLab.String())
	} else {
		c.parseArgs(&rackRegistrationReq)
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

	res, err := ic.RackRegistration(ctx, &rackRegistrationReq)
	if err != nil {
		return err
	}
	utils.PrintProtoJSON(res)
	fmt.Println("Successfully added the rack: ", rackRegistrationReq.GetRack().GetName())
	return nil
}

func (c *addRack) parseArgs(req *ufsAPI.RackRegistrationRequest) {
	ufsLab := utils.ToUFSLab(c.labName)
	req.Rack = &ufspb.Rack{
		Name: c.rackName,
		Location: &ufspb.Location{
			Lab: ufsLab,
		},
		CapacityRu: int32(c.capacity),
		Realm:      utils.ToUFSRealm(c.labName),
	}
	if ufsUtil.IsInBrowserLab(ufsLab.String()) {
		req.Rack.Rack = &ufspb.Rack_ChromeBrowserRack{
			ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
		}
	} else {
		req.Rack.Rack = &ufspb.Rack_ChromeosRack{
			ChromeosRack: &ufspb.ChromeOSRack{},
		}
	}
}

func (c *addRack) validateArgs() error {
	if c.labName == "" {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\n'-lab' is required.")
	}
	if !utils.IsUFSLab(c.labName) {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\n%s is not a valid lab name, please check help info for '-lab'.", c.labName)
	}
	if c.newSpecsFile != "" {
		if c.rackName != "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-name' cannot be specified at the same time.")
		}
	} else {
		if c.rackName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f') is setup, so '-name' is required.")
		}
	}
	return nil
}
