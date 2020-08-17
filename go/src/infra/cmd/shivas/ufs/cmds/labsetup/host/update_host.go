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

// UpdateHostCmd update a host on a machine.
var UpdateHostCmd = &subcommands.Command{
	UsageLine: "update-host [Options...]",
	ShortDesc: "Update a host(DUT, Labstation, Dev Server, Caching Server, VM Server, Host OS...) on a machine",
	LongDesc:  cmdhelp.UpdateHostLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateHost{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.MachineLSEFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.machineName, "machine", "", "name of the machine to associate the host")
		c.Flags.StringVar(&c.vlanName, "vlan", "", "name of the vlan to assign this host to")
		c.Flags.StringVar(&c.nicName, "nic", "", "name of the nic to associate the ip to")
		c.Flags.StringVar(&c.hostName, "name", "", "name of the host")
		c.Flags.BoolVar(&c.deleteVlan, "delete-vlan", false, "if deleting the ip assignment for the host")
		c.Flags.StringVar(&c.ip, "ip", "", "the ip to assign the host to")
		c.Flags.StringVar(&c.state, "state", "", cmdhelp.StateHelp)
		c.Flags.StringVar(&c.prototype, "prototype", "", "name of the prototype to be used to deploy this host.")
		c.Flags.StringVar(&c.osVersion, "os-version", "", "name of the os version of the machine (browser lab only). "+cmdhelp.ClearFieldHelpText)
		c.Flags.IntVar(&c.vmCapacity, "vm-capacity", 0, "the number of the vms that this machine supports (browser lab only). "+"To clear this field set it to -1.")
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags. You can only append/add new tags here. "+cmdhelp.ClearFieldHelpText)
		return c
	},
}

type updateHost struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string
	interactive  bool

	machineName string
	hostName    string
	vlanName    string
	nicName     string
	deleteVlan  bool
	ip          string
	state       string
	prototype   string
	osVersion   string
	vmCapacity  int
	tags        string
}

func (c *updateHost) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateHost) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	machinelse := &ufspb.MachineLSE{}
	if c.interactive {
		return errors.New("Interactive mode for this " +
			"command is not yet implemented yet. Use JSON input mode.")
		//TODO(eshwarn): add interactive input
		//utils.GetMachinelseInteractiveInput(ctx, ic, &machinelse, true)
	}
	if c.newSpecsFile != "" {
		if err = utils.ParseJSONFile(c.newSpecsFile, machinelse); err != nil {
			return err
		}
	} else {
		c.parseArgs(machinelse)
	}

	var networkOptions map[string]*ufsAPI.NetworkOption
	var states map[string]ufspb.State
	if c.deleteVlan || c.vlanName != "" || c.ip != "" {
		networkOptions = map[string]*ufsAPI.NetworkOption{
			machinelse.Name: {
				Delete: c.deleteVlan,
				Vlan:   c.vlanName,
				Nic:    c.nicName,
				Ip:     c.ip,
			},
		}
	}
	if c.state != "" {
		states = map[string]ufspb.State{
			machinelse.Name: ufsUtil.ToUFSState(c.state),
		}
	}

	var machineNames []string
	if c.machineName != "" {
		machineNames = append(machineNames, c.machineName)
	}
	machinelse.Name = ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, machinelse.Name)
	res, err := ic.UpdateMachineLSE(ctx, &ufsAPI.UpdateMachineLSERequest{
		MachineLSE:     machinelse,
		Machines:       machineNames,
		NetworkOptions: networkOptions,
		States:         states,
		UpdateMask: utils.GetUpdateMask(&c.Flags, map[string]string{
			"machine":     "machine",
			"prototype":   "mlseprototype",
			"os-version":  "osVersion",
			"vm-capacity": "vmCapacity",
			"tags":        "tags",
			"state":       "state",
		}),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	if c.deleteVlan {
		fmt.Printf("Successfully deleted vlan of host %s\n", res.Name)
	}
	if c.vlanName != "" || c.ip != "" {
		// Log the assigned IP
		if dhcp, err := ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
			Hostname: res.Name,
		}); err == nil {
			utils.PrintProtoJSON(dhcp)
			fmt.Println("Successfully added dhcp config to host: ", res.Name)
		}
	}
	return nil
}

func (c *updateHost) parseArgs(lse *ufspb.MachineLSE) {
	lse.Name = c.hostName
	lse.Hostname = c.hostName
	lse.MachineLsePrototype = c.prototype
	if c.tags == utils.ClearFieldValue {
		lse.Tags = nil
	} else {
		lse.Tags = utils.GetStringSlice(c.tags)
	}
	if c.osVersion != "" || c.vmCapacity != 0 {
		lse.Lse = &ufspb.MachineLSE_ChromeBrowserMachineLse{
			ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{
				OsVersion: &ufspb.OSVersion{},
			},
		}
		if c.vmCapacity == -1 {
			lse.GetChromeBrowserMachineLse().VmCapacity = 0
		} else {
			lse.GetChromeBrowserMachineLse().VmCapacity = int32(c.vmCapacity)
		}
		if c.osVersion == utils.ClearFieldValue {
			lse.GetChromeBrowserMachineLse().GetOsVersion().Value = ""
		} else {
			lse.GetChromeBrowserMachineLse().GetOsVersion().Value = c.osVersion
		}
	}
}

func (c *updateHost) validateArgs() error {
	if c.newSpecsFile != "" && c.interactive {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.hostName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f' or '-i') is specified.")
		}
		if c.vlanName == "" && !c.deleteVlan && c.ip == "" && c.state == "" &&
			c.osVersion == "" && c.prototype == "" && c.tags == "" && c.vmCapacity == 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNothing to update. Please provide any field to update")
		}
	}
	if c.state != "" && !ufsUtil.IsUFSState(ufsUtil.RemoveStatePrefix(c.state)) {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid state, please check help info for '-state'.", c.state)
	}
	return nil
}
