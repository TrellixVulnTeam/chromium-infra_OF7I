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
		c.Flags.StringVar(&c.vlanName, "vlan", "", "name of the vlan to assign this host to")
		c.Flags.StringVar(&c.nicName, "nic", "", "name of the nic to associate the ip to")

		c.Flags.StringVar(&c.hostName, "name", "", "name of the host")
		c.Flags.StringVar(&c.prototype, "prototype", "", "name of the prototype to be used to deploy this host")
		c.Flags.StringVar(&c.osVersion, "os-version", "", "name of the os version of the machine (browser lab only)")
		c.Flags.IntVar(&c.vmCapacity, "vm-capacity", 0, "the number of the vms that this machine supports (browser lab only)")
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags. You can only append/add new tags here.")
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
	vlanName    string
	nicName     string
	hostName    string
	prototype   string
	osVersion   string
	vmCapacity  int
	tags        string
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

	req := &ufsAPI.CreateMachineLSERequest{
		MachineLSE:   &machinelse,
		MachineLSEId: machinelse.GetName(),
		Machines:     []string{c.machineName},
	}
	if c.vlanName != "" && c.nicName != "" {
		req.NetworkOption = &ufsAPI.NetworkOption{
			Vlan: c.vlanName,
			Nic:  c.nicName,
		}
	}

	res, err := ic.CreateMachineLSE(ctx, req)
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	fmt.Println("Successfully added the host: ", machinelse.GetName())
	if c.vlanName != "" && c.nicName != "" {
		// Log the assigned IP
		if dhcp, err := ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
			Hostname: res.GetHostname(),
		}); err == nil {
			utils.PrintProtoJSON(dhcp)
			fmt.Println("Successfully added dhcp config to host: ", machinelse.GetName())
		}
	}
	return nil
}

func (c *addHost) parseArgs(lse *ufspb.MachineLSE, ufsLab ufspb.Lab) {
	lse.Hostname = c.hostName
	lse.Name = c.hostName
	lse.MachineLsePrototype = c.prototype
	lse.Tags = utils.GetStringSlice(c.tags)
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
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.prototype != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-prototype' cannot be specified at the same time.")
		}
		if c.tags != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-tags' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile != "" {
		if c.interactive {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
		}
		if c.machineName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nMachine name (-machine) is required for JSON mode.")
		}
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.hostName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f' or '-i') is specified.")
		}
		if c.machineName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-machine' is required, no mode ('-f' or '-i') is specified.")
		}
		if c.prototype == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-prototype' is required, no mode ('-f' or '-i') is specified. Please run `shivas list machine-prototype` to check valid prototypes for your host")
		}
	}
	return nil
}
