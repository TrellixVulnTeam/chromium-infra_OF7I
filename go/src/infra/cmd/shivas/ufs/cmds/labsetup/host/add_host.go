// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package host

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
	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// AddHostCmd add a host to the machine.
var AddHostCmd = &subcommands.Command{
	UsageLine: "add-host [Options..]",
	ShortDesc: "Add a host(DUT, Labstation, Dev Server, Caching Server, VM Server, Host OS...) on a machine",
	LongDesc:  cmdhelp.AddHostLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addHost{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.MachineLSEFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.machineName, "machine", "", "name of the machine to associate the host")
		c.Flags.StringVar(&c.hostName, "name", "", "name of the host")
		c.Flags.StringVar(&c.prototype, "prototype", "", "name of the prototype to be used to deploy this host")
		c.Flags.StringVar(&c.osVersion, "os-version", "", "name of the os version of the machine (browser lab only)")
		c.Flags.IntVar(&c.vmCapacity, "vm-capacity", 0, "the number of the vms that this machine supports (browser lab only)")
		return c
	},
}

type addHost struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string
	interactive  bool

	machineName string
	hostName    string
	prototype   string
	osVersion   string
	vmCapacity  int
}

func (c *addHost) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addHost) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	var machinelse ufspb.MachineLSE
	if c.interactive {
		return errors.New("Interactive mode for this " +
			"command is not yet implemented yet. Use JSON input mode.")
		//TODO(eshwarn): add interactive input
		//utils.GetMachinelseInteractiveInput(ctx, ic, &machinelse, false)
	}
	if c.newSpecsFile != "" {
		if err = utils.ParseJSONFile(c.newSpecsFile, &machinelse); err != nil {
			return err
		}
	} else {
		machine, err := ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, c.machineName)})
		if err != nil {
			return errors.New(fmt.Sprintf("Fail to find machine %s", c.machineName))
		}
		c.parseArgs(&machinelse, machine.GetLocation().GetLab())
	}

	res, err := ic.CreateMachineLSE(ctx, &ufsAPI.CreateMachineLSERequest{
		MachineLSE:   &machinelse,
		MachineLSEId: machinelse.GetName(),
		Machines:     []string{c.machineName},
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	fmt.Println("Successfully added the host: ", machinelse.GetName())
	return nil
}

func (c *addHost) parseArgs(lse *ufspb.MachineLSE, ufsLab ufspb.Lab) {
	lse.Hostname = c.hostName
	lse.Name = c.hostName
	lse.MachineLsePrototype = c.prototype
	if ufsUtil.IsInBrowserLab(ufsLab.String()) {
		lse.Lse = &ufspb.MachineLSE_ChromeBrowserMachineLse{
			ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{
				OsVersion: &ufspb.OSVersion{
					Value: c.osVersion,
				},
				VmCapacity: int32(c.vmCapacity),
			},
		}
	} else {
		lse.Lse = &ufspb.MachineLSE_ChromeosMachineLse{
			ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{},
		}
	}
}

func (c *addHost) validateArgs() error {
	if c.newSpecsFile != "" || c.interactive {
		if c.hostName != "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.prototype != "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-prototype' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile != "" {
		if c.interactive {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
		}
		if c.machineName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nMachine name (-machine) is required for JSON mode.")
		}
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.hostName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so '-name' is required.")
		}
		if c.machineName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so '-machine' is required.")
		}
		if c.prototype == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so '-prototype' is required.")
		}
	}
	return nil
}
