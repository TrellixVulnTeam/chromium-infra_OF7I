// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rack

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
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// AddRackCmd add Rack to the system.
var AddRackCmd = &subcommands.Command{
	UsageLine: "rack [Options...]",
	ShortDesc: "Add a rack",
	LongDesc:  cmdhelp.AddRackLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addRack{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.RackRegistrationFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.rackName, "name", "", "the name of the rack to add")
		c.Flags.StringVar(&c.zoneName, "zone", "", cmdhelp.ZoneHelpText)
		c.Flags.IntVar(&c.capacity, "capacity_ru", 0, "indicate the size of the rack in rack units (U).")
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags. You can only append/add new tags here.")
		return c
	},
}

type addRack struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string
	interactive  bool

	rackName string
	zoneName string
	capacity int
	tags     string
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
	var rackRegistrationReq ufsAPI.RackRegistrationRequest
	if c.interactive {
		utils.GetRackInteractiveInput(ctx, ic, &rackRegistrationReq)
	} else if c.newSpecsFile != "" {
		if err := utils.ParseJSONFile(c.newSpecsFile, &rackRegistrationReq); err != nil {
			return err
		}
		ufsZone := rackRegistrationReq.GetRack().GetLocation().GetZone()
		rackRegistrationReq.GetRack().Realm = ufsUtil.ToUFSRealm(ufsZone.String())
	} else {
		c.parseArgs(&rackRegistrationReq)
	}
	res, err := ic.RackRegistration(ctx, &rackRegistrationReq)
	if err != nil {
		return err
	}
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Println("Successfully added the rack: ", res.GetName())
	return nil
}

func (c *addRack) parseArgs(req *ufsAPI.RackRegistrationRequest) {
	ufsZone := ufsUtil.ToUFSZone(c.zoneName)
	req.Rack = &ufspb.Rack{
		Name: c.rackName,
		Location: &ufspb.Location{
			Zone: ufsZone,
		},
		CapacityRu: int32(c.capacity),
		Realm:      ufsUtil.ToUFSRealm(c.zoneName),
		Tags:       utils.GetStringSlice(c.tags),
	}
	if ufsUtil.IsInBrowserZone(ufsZone.String()) {
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
	if c.newSpecsFile != "" && c.interactive {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
	}
	if c.newSpecsFile != "" || c.interactive {
		if c.rackName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-name' cannot be specified at the same time.")
		}
		if c.zoneName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-zone' cannot be specified at the same time.")
		}
		if c.capacity != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-capacity_ru' cannot be specified at the same time.")
		}
		if c.tags != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-tags' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.rackName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f') is setup.")
		}
		if c.zoneName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-zone' is required, no mode ('-f') is setup.")
		}
		if !ufsUtil.IsUFSZone(ufsUtil.RemoveZonePrefix(c.zoneName)) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid zone name, please check help info for '-zone'.", c.zoneName)
		}
	}
	return nil
}
