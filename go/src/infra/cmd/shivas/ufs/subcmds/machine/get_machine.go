// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package machine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/jsonpb"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// GetMachineCmd get machine by given name.
var GetMachineCmd = &subcommands.Command{
	UsageLine: "machine {Machine name}",
	ShortDesc: "Get machine details by name",
	LongDesc: `Get machine details by name.

Example:

shivas get machine {Machine name}
Gets the machine and prints the output in JSON format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getMachine{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)
		return c
	},
}

type getMachine struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags
}

func (c *getMachine) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getMachine) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
		fmt.Printf("Using UnifiedFleet service %s\n", e.UnifiedFleetService)
	}
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	machine, err := ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, args[0]),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get machine").Err()
	}
	machine.Name = ufsUtil.RemovePrefix(machine.Name)
	if c.outputFlags.Full() {
		return c.printFull(ctx, ic, machine)
	}
	return c.print(machine)
}

func (c *getMachine) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the machine name or deployed machine hostname.")
	}
	return nil
}

func (c *getMachine) printFull(ctx context.Context, ic ufsAPI.FleetClient, machine *ufspb.Machine) error {
	req := &ufsAPI.ListMachineLSEsRequest{
		Filter: ufsUtil.MachineFilterName + "=" + machine.Name,
	}
	var lse *ufspb.MachineLSE
	res, err := ic.ListMachineLSEs(ctx, req)
	if err != nil {
		if c.commonFlags.Verbose() {
			fmt.Printf("Failed to get host for the machine: %s", err)
		}
	} else {
		if c.commonFlags.Verbose() && len(res.MachineLSEs) > 1 {
			fmt.Printf("More than one host associated with this machine. Data discrepancy warning.\n%s\n", res.MachineLSEs)
		}
		if len(res.GetMachineLSEs()) > 0 {
			lse = res.GetMachineLSEs()[0]
			lse.Name = ufsUtil.RemovePrefix(lse.Name)
		}
	}
	rack, err := ic.GetRack(ctx, &ufsAPI.GetRackRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.RackCollection, machine.GetLocation().GetRack()),
	})
	if c.outputFlags.JSON() {
		return printMachineJSONFull(machine, lse, rack)
	}
	if !c.outputFlags.Tsv() {
		if machine.GetChromeBrowserMachine() != nil {
			utils.PrintTitle(utils.BrowserMachineFullTitle)
		} else if machine.GetChromeosMachine() != nil {
			utils.PrintTitle(utils.OSMachineFullTitle)
		}
	}
	utils.PrintMachineFull(machine, lse, rack)
	return nil
}

func (c *getMachine) print(machine *ufspb.Machine) error {
	if c.outputFlags.JSON() {
		utils.PrintProtoJSON(machine)
	} else {
		if !c.outputFlags.Tsv() {
			if machine.GetChromeBrowserMachine() != nil {
				utils.PrintTitle(utils.MachineTitle)
			} else if machine.GetChromeosMachine() != nil {
				utils.PrintTitle(utils.OSMachineFullTitle)
			}
		}
		utils.PrintMachines([]*ufspb.Machine{machine}, false)
	}
	return nil
}

func printMachineJSONFull(machine *ufspb.Machine, machinelse *ufspb.MachineLSE, rack *ufspb.Rack) error {
	jm := jsonpb.Marshaler{
		EnumsAsInts: false,
		Indent:      "\t",
	}
	machineJSON, err := jm.MarshalToString(machine)
	if err != nil {
		return errors.Annotate(err, "Failed to marshal machineJSON").Err()
	}
	machinelseJSON, err := jm.MarshalToString(machinelse)
	if err != nil {
		return errors.Annotate(err, "Failed to marshal machineLSEJSON").Err()
	}
	machineout := map[string]interface{}{}
	if err := json.Unmarshal([]byte(machineJSON), &machineout); err != nil {
		return errors.Annotate(err, "Failed to unmarshal machineJSON").Err()
	}
	machinelseout := map[string]interface{}{}
	if err := json.Unmarshal([]byte(machinelseJSON), &machinelseout); err != nil {
		return errors.Annotate(err, "Failed to unmarshal machinelseJSON").Err()
	}
	machineout["host"] = machinelseout

	rackJSON, err := jm.MarshalToString(rack)
	if err != nil {
		return errors.Annotate(err, "Failed to marshal rackJSON").Err()
	}
	rackout := map[string]interface{}{}
	if err := json.Unmarshal([]byte(rackJSON), &rackout); err != nil {
		return errors.Annotate(err, "Failed to unmarshal rackJSON").Err()
	}
	machineout["rack"] = rackout

	outputJSON, err := json.MarshalIndent(machineout, "", "\t")
	if err != nil {
		return errors.Annotate(err, "Failed to marshal final machine output").Err()
	}
	fmt.Printf("%s", outputJSON)
	fmt.Println()
	return nil
}
