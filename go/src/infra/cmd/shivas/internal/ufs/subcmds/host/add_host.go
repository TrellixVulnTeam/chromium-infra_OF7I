// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package host

import (
	"context"
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

// AddHostCmd add a host to the machine.
var AddHostCmd = &subcommands.Command{
	UsageLine: "host [Options..]",
	ShortDesc: "Add a host(Dev Server, VM Server, Host OS...) on a machine",
	LongDesc:  cmdhelp.AddHostLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addHost{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.MachineLSEFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.vlanName, "vlan", "", "name of the vlan to assign this host to")
		c.Flags.StringVar(&c.nicName, "nic", "", "name of the nic to associate the ip to")
		c.Flags.StringVar(&c.ip, "ip", "", "the ip to assign the host to")

		c.Flags.StringVar(&c.hostName, "name", "", "name of the host")
		c.Flags.StringVar(&c.machineName, "machine", "", "name of the machine to associate the host")
		c.Flags.StringVar(&c.prototype, "prototype", "", "name of the prototype to be used to deploy this host")
		c.Flags.StringVar(&c.osVersion, "os", "", "name of the os version of the machine (browser lab only)")
		c.Flags.IntVar(&c.vmCapacity, "vm-capacity", 0, "the number of the vms that this machine supports (browser lab only)")
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags. You can only append/add new tags here.")
		c.Flags.StringVar(&c.deploymentTicket, "ticket", "", "the deployment ticket for this host")
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

	machineName      string
	vlanName         string
	nicName          string
	ip               string
	hostName         string
	prototype        string
	osVersion        string
	vmCapacity       int
	tags             string
	deploymentTicket string
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
		if machinelse.GetMachines() == nil || len(machinelse.GetMachines()) <= 0 {
			return errors.New(fmt.Sprintf("machines field is empty in json. It is a required parameter for json input."))
		}
	} else {
		machine, err := ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, c.machineName)})
		if err != nil {
			return errors.New(fmt.Sprintf("Fail to find machine %s", c.machineName))
		}
		c.parseArgs(&machinelse, machine.GetLocation().GetZone())
	}

	if !ufsUtil.ValidateTags(machinelse.Tags) {
		return fmt.Errorf(ufsAPI.InvalidTags)
	}

	req := &ufsAPI.CreateMachineLSERequest{
		MachineLSE:    &machinelse,
		MachineLSEId:  machinelse.GetName(),
		NetworkOption: c.parseNetworkOpt(),
	}

	res, err := ic.CreateMachineLSE(ctx, req)
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	c.printRes(ctx, ic, res)
	return nil
}

func (c *addHost) printRes(ctx context.Context, ic ufsAPI.FleetClient, res *ufspb.MachineLSE) {
	fmt.Println("The newly added host:")
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	if c.vlanName != "" && c.nicName != "" {
		// Log the assigned IP
		if dhcp, err := ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
			Hostname: res.GetHostname(),
		}); err == nil {
			fmt.Println("Newly added DHCP config:")
			utils.PrintProtoJSON(dhcp, false)
			fmt.Printf("Successfully added dhcp config %s to vm %s\nPlease run `shivas get host -full %s` to further check\n", dhcp.GetIp(), res.Name, res.Name)
		}
	}
}

func (c *addHost) parseArgs(lse *ufspb.MachineLSE, ufsZone ufspb.Zone) {
	lse.Hostname = c.hostName
	lse.Name = c.hostName
	lse.MachineLsePrototype = c.prototype
	lse.Tags = utils.GetStringSlice(c.tags)
	lse.Machines = []string{c.machineName}
	lse.DeploymentTicket = c.deploymentTicket
	if ufsUtil.IsInBrowserZone(ufsZone.String()) {
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

func (c *addHost) parseNetworkOpt() *ufsAPI.NetworkOption {
	if c.ip != "" || c.vlanName != "" {
		fmt.Println("Setting network option parameters")
		if c.ip != "" {
			return &ufsAPI.NetworkOption{
				Ip:  c.ip,
				Nic: c.nicName,
			}
		}
		if c.vlanName != "" {
			return &ufsAPI.NetworkOption{
				Vlan: c.vlanName,
				Nic:  c.nicName,
			}
		}
	}
	return nil
}

func (c *addHost) validateArgs() error {
	if c.newSpecsFile != "" && c.interactive {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
	}
	if c.newSpecsFile != "" || c.interactive {
		if c.hostName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.machineName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-machine' cannot be specified at the same time.")
		}
		if c.prototype != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-prototype' cannot be specified at the same time.")
		}
		if c.tags != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-tags' cannot be specified at the same time.")
		}
		if c.vmCapacity != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-vm-capacity' cannot be specified at the same time.")
		}
		if c.osVersion != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-os' cannot be specified at the same time.")
		}
		if c.deploymentTicket != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-ticket' cannot be specified at the same time.")
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
