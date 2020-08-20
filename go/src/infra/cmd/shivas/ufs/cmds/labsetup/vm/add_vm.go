// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package vm

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

// AddVMCmd add a vm on a host.
var AddVMCmd = &subcommands.Command{
	UsageLine: "add-vm [Options..]",
	ShortDesc: "Add a VM on a host",
	LongDesc: `Add a VM on a host

Examples:
shivas add-vm -new-json-file vm.json -host host1
Add a VM on a host by reading a JSON file input.

shivas add-vm -name vm1 -host host1 -mac-address 12:34:56 -os chrome-version-1
Add a VM by parameters.`,
	CommandRun: func() subcommands.CommandRun {
		c := &addVM{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.VMFileText)

		c.Flags.StringVar(&c.hostName, "host", "", "hostname of the host to add the VM")
		c.Flags.StringVar(&c.vmName, "name", "", "hostname/name of the VM")
		c.Flags.StringVar(&c.macAddress, "mac-address", "", "mac address of the VM")
		c.Flags.StringVar(&c.osVersion, "os", "", "os version of the VM")
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags. You can only append/add new tags here.")
		return c
	},
}

type addVM struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string

	hostName   string
	vmName     string
	macAddress string
	osVersion  string
	tags       string
}

func (c *addVM) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addVM) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	// Parse input json
	var vm ufspb.VM
	if c.newSpecsFile != "" {
		if err = utils.ParseJSONFile(c.newSpecsFile, &vm); err != nil {
			return err
		}
	} else {
		c.parseArgs(&vm)
	}

	res, err := ic.CreateVM(ctx, &ufsAPI.CreateVMRequest{
		Vm:           &vm,
		MachineLSEId: c.hostName,
	})
	if err != nil {
		return errors.Annotate(err, "Unable to add the VM to the host").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res, false)
	fmt.Printf("Successfully added the vm %s to host %s\n", vm.GetName(), c.hostName)
	return nil
}

func (c *addVM) parseArgs(vm *ufspb.VM) {
	vm.Name = c.vmName
	vm.Hostname = c.vmName
	vm.MacAddress = c.macAddress
	vm.OsVersion = &ufspb.OSVersion{
		Value: c.osVersion,
	}
	vm.Tags = utils.GetStringSlice(c.tags)
}

func (c *addVM) validateArgs() error {
	if c.hostName == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-host' is required.")
	}
	if c.newSpecsFile != "" {
		if c.vmName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.macAddress != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-mac-address' cannot be specified at the same time.")
		}
		if c.tags != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-tags' cannot be specified at the same time.")
		}
	} else {
		if c.vmName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f') is specified.")
		}
		if c.macAddress == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-mac-address' is required, no mode ('-f') is specified.")
		}
	}
	return nil
}
