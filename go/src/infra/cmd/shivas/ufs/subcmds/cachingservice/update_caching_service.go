// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cachingservice

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

// UpdateCachingServiceCmd Update CachingService by given name.
var UpdateCachingServiceCmd = &subcommands.Command{
	UsageLine: "CachingService [Options...]",
	ShortDesc: "Update a CachingService",
	LongDesc:  cmdhelp.UpdateCachingServiceLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateCachingService{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.CachingServiceUpdateFileText)

		c.Flags.StringVar(&c.name, "name", "", "name of the CachingService")
		c.Flags.IntVar(&c.port, "port", defaultCachingServicePort, "port number of the CachingService. "+"To set it to default port 8082, enter -1('-port -1').")
		c.Flags.Var(utils.CSVString(&c.subnets), "subnets", "append/clear comma separated subnet list which this CachingService serves/supports. "+cmdhelp.ClearFieldHelpText)
		c.Flags.Var(utils.CSVString(&c.removeSubnets), "subnets-to-remove", "remove comma separated subnet list.")
		c.Flags.StringVar(&c.primary, "primary", "", "primary node ip of the CachingService. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.secondary, "secondary", "", "secondary node ip of the CachingService. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.state, "state", "", cmdhelp.StateHelp)
		c.Flags.StringVar(&c.description, "desc", "", "description for the CachingService. "+cmdhelp.ClearFieldHelpText)
		return c
	},
}

type updateCachingService struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string

	name          string
	port          int
	subnets       []string
	removeSubnets []string
	primary       string
	secondary     string
	state         string
	description   string
}

func (c *updateCachingService) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateCachingService) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	var cs ufspb.CachingService
	if c.newSpecsFile != "" {
		if err = utils.ParseJSONFile(c.newSpecsFile, &cs); err != nil {
			return err
		}
	} else {
		c.parseArgs(&cs)
	}
	if err := utils.PrintExistingCachingService(ctx, ic, cs.Name); err != nil {
		return err
	}
	cs.Name = ufsUtil.AddPrefix(ufsUtil.CachingServiceCollection, cs.Name)
	res, err := ic.UpdateCachingService(ctx, &ufsAPI.UpdateCachingServiceRequest{
		CachingService: &cs,
		UpdateMask: utils.GetUpdateMask(&c.Flags, map[string]string{
			"port":              "port",
			"subnets":           "serving_subnets",
			"subnets-to-remove": "serving_subnets.remove",
			"primary":           "primary_node",
			"secondary":         "secondary_node",
			"state":             "state",
			"desc":              "description",
		}),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The CachingService after update:")
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Printf("Successfully updated the CachingService %s\n", res.Name)
	return nil
}

func (c *updateCachingService) parseArgs(cs *ufspb.CachingService) {
	cs.Name = c.name
	if c.port == -1 {
		cs.Port = int32(defaultCachingServicePort)
	} else {
		cs.Port = int32(c.port)
	}
	if len(c.removeSubnets) > 0 {
		cs.ServingSubnets = c.removeSubnets
	} else if ufsUtil.ContainsAnyStrings(c.subnets, utils.ClearFieldValue) {
		cs.ServingSubnets = nil
	} else {
		cs.ServingSubnets = c.subnets
	}
	if c.primary == utils.ClearFieldValue {
		cs.PrimaryNode = ""
	} else {
		cs.PrimaryNode = c.primary
	}
	if c.secondary == utils.ClearFieldValue {
		cs.SecondaryNode = ""
	} else {
		cs.SecondaryNode = c.secondary
	}
	cs.State = ufsUtil.ToUFSState(c.state)
	if c.description == utils.ClearFieldValue {
		cs.Description = ""
	} else {
		cs.Description = c.description
	}
}

func (c *updateCachingService) validateArgs() error {
	if c.newSpecsFile != "" {
		if c.name != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.port != defaultCachingServicePort {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-port' cannot be specified at the same time.")
		}
		if len(c.subnets) != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-subnets' cannot be specified at the same time.")
		}
		if len(c.removeSubnets) != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-subnets-to-remove' cannot be specified at the same time.")
		}
		if c.primary != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-primary' cannot be specified at the same time.")
		}
		if c.secondary != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-secondary' cannot be specified at the same time.")
		}
		if c.state != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-state' cannot be specified at the same time.")
		}
		if c.description != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe file mode is specified. '-description' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile == "" {
		if c.name == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f') is specified.")
		}
		if len(c.subnets) != 0 && len(c.removeSubnets) != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-subnets' and '-subnets-to-remove' cannot be specified at the same time.")
		}
		if len(c.subnets) == 0 && c.port == defaultCachingServicePort && c.description == "" && c.primary == "" && c.secondary == "" && c.state == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNothing to update. Please provide any field to update")
		}
		if c.state != "" && !ufsUtil.IsUFSState(ufsUtil.RemoveStatePrefix(c.state)) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid state, please check help info for '-state'.", c.state)
		}
	}
	return nil
}
