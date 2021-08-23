// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package schedulingunit

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

// UpdateSchedulingUnitCmd Update SchedulingUnit by given name.
var UpdateSchedulingUnitCmd = &subcommands.Command{
	UsageLine: "SchedulingUnit [Options...]",
	ShortDesc: "Update a SchedulingUnit",
	LongDesc:  cmdhelp.UpdateSchedulingUnitLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateSchedulingUnit{
			pools:       []string{},
			duts:        []string{},
			tags:        []string{},
			removePools: []string{},
			removeDuts:  []string{},
			removeTags:  []string{},
		}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.SchedulingUnitUpdateFileText)

		c.Flags.StringVar(&c.name, "name", "", "name of the SchedulingUnit")
		c.Flags.Var(utils.CSVString(&c.pools), "pools", "append/clear comma separated pools. "+cmdhelp.ClearFieldHelpText)
		c.Flags.Var(utils.CSVString(&c.duts), "duts", "append/clear comma separated DUTs. "+cmdhelp.ClearFieldHelpText)
		c.Flags.Var(utils.CSVString(&c.tags), "tags", "append/clear comma separated tags. "+cmdhelp.ClearFieldHelpText)
		c.Flags.Var(utils.CSVString(&c.removePools), "pools-to-remove", "remove comma separated pools.")
		c.Flags.Var(utils.CSVString(&c.removeDuts), "duts-to-remove", "remove comma separated DUTs.")
		c.Flags.Var(utils.CSVString(&c.removeTags), "tags-to-remove", "remove comma separated tags.")
		c.Flags.StringVar(&c.schedulingUnitType, "type", "", "Type of SchedulingUnit. "+cmdhelp.SchedulingUnitTypesHelpText)
		c.Flags.StringVar(&c.description, "desc", "", "description for the SchedulingUnit")
		return c
	},
}

type updateSchedulingUnit struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string

	name               string
	pools              []string
	duts               []string
	tags               []string
	removePools        []string
	removeDuts         []string
	removeTags         []string
	schedulingUnitType string
	description        string
}

func (c *updateSchedulingUnit) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateSchedulingUnit) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	ctx = utils.SetupContext(ctx, ufsUtil.OSNamespace)
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
	var su ufspb.SchedulingUnit
	if c.newSpecsFile != "" {
		if err = utils.ParseJSONFile(c.newSpecsFile, &su); err != nil {
			return err
		}
	} else {
		c.parseArgs(&su)
	}
	if err := utils.PrintExistingSchedulingUnit(ctx, ic, su.Name); err != nil {
		return err
	}
	su.Name = ufsUtil.AddPrefix(ufsUtil.SchedulingUnitCollection, su.Name)
	res, err := ic.UpdateSchedulingUnit(ctx, &ufsAPI.UpdateSchedulingUnitRequest{
		SchedulingUnit: &su,
		UpdateMask: utils.GetUpdateMask(&c.Flags, map[string]string{
			"duts":            "machinelses",
			"pools":           "pools",
			"tags":            "tags",
			"duts-to-remove":  "machinelses.remove",
			"pools-to-remove": "pools.remove",
			"tags-to-remove":  "tags.remove",
			"type":            "type",
			"desc":            "description",
		}),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The SchedulingUnit after update:")
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Printf("Successfully updated the SchedulingUnit %s\n", res.Name)
	return nil
}

func (c *updateSchedulingUnit) parseArgs(su *ufspb.SchedulingUnit) {
	su.Name = c.name
	if len(c.removeDuts) > 0 {
		su.MachineLSEs = c.removeDuts
	} else if ufsUtil.ContainsAnyStrings(c.duts, utils.ClearFieldValue) {
		su.MachineLSEs = nil
	} else {
		su.MachineLSEs = c.duts
	}
	if len(c.removePools) > 0 {
		su.Pools = c.removePools
	} else if ufsUtil.ContainsAnyStrings(c.pools, utils.ClearFieldValue) {
		su.Pools = nil
	} else {
		su.Pools = c.pools
	}
	if len(c.removeTags) > 0 {
		su.Tags = c.removeTags
	} else if ufsUtil.ContainsAnyStrings(c.tags, utils.ClearFieldValue) {
		su.Tags = nil
	} else {
		su.Tags = c.tags
	}
	su.Type = ufsUtil.ToSchedulingUnitType(c.schedulingUnitType)
	if c.description == utils.ClearFieldValue {
		su.Description = ""
	} else {
		su.Description = c.description
	}
}

func (c *updateSchedulingUnit) validateArgs() error {
	if c.newSpecsFile != "" {
		if c.name != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-name' cannot be specified at the same time.")
		}
		if len(c.pools) != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-pools' cannot be specified at the same time.")
		}
		if len(c.duts) != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-duts' cannot be specified at the same time.")
		}
		if len(c.tags) != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-tags' cannot be specified at the same time.")
		}
		if len(c.removePools) != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-pools-to-remove' cannot be specified at the same time.")
		}
		if len(c.removeDuts) != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-duts-to-remove' cannot be specified at the same time.")
		}
		if len(c.removeTags) != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-tags-to-remove' cannot be specified at the same time.")
		}
		if c.schedulingUnitType != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-type' cannot be specified at the same time.")
		}
		if c.description != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-description' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile == "" {
		if c.name == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f') is specified.")
		}
		if c.schedulingUnitType != "" && !ufsUtil.IsSchedulingUnitType(c.schedulingUnitType) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid SchedulingUnitType name, please check help info for '-type'.", c.schedulingUnitType)
		}
		if len(c.duts) != 0 && len(c.removeDuts) != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-duts' and '-duts-to-remove' cannot be specified at the same time.")
		}
		if len(c.pools) != 0 && len(c.removePools) != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-pools' and '-pools-to-remove' cannot be specified at the same time.")
		}
		if len(c.tags) != 0 && len(c.removeTags) != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-tags' and '-tags-to-remove' cannot be specified at the same time.")
		}
		if c.name == "" && c.schedulingUnitType == "" && c.description == "" && len(c.duts) == 0 && len(c.removeDuts) == 0 &&
			len(c.tags) == 0 && len(c.removeTags) == 0 && len(c.pools) == 0 && len(c.removePools) == 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNothing to update. Please provide any field to update")
		}
	}
	return nil
}
