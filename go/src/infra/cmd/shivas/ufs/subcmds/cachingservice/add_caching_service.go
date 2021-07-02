// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cachingservice

import (
	"fmt"
	"strconv"
	"strings"

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

// defaultCachingServicePort is the default port for the CachingService.
const defaultCachingServicePort = 8082

// AddCachingServiceCmd add CachingService to the system.
var AddCachingServiceCmd = &subcommands.Command{
	UsageLine: "cachingservice",
	ShortDesc: "Add CachingService",
	LongDesc:  cmdhelp.AddCachingServiceLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addCachingService{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.CachingServiceFileText)

		c.Flags.StringVar(&c.name, "name", "", "name of the CachingService")
		c.Flags.IntVar(&c.port, "port", defaultCachingServicePort, "port number of the CachingService")
		c.Flags.Var(utils.CSVString(&c.subnets), "subnets", "comma separated subnet list which this CachingService serves/supports")
		c.Flags.StringVar(&c.primary, "primary", "", "primary node ip of the CachingService")
		c.Flags.StringVar(&c.secondary, "secondary", "", "secondary node ip of the CachingService")
		c.Flags.StringVar(&c.state, "state", "", cmdhelp.StateHelp)
		c.Flags.StringVar(&c.description, "desc", "", "description for the CachingService")
		return c
	},
}

type addCachingService struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string

	name        string
	port        int
	subnets     []string
	primary     string
	secondary   string
	state       string
	description string
}

var mcsvFields = []string{
	"name",
	"port",
	"subnets",
	"primary",
	"secondary",
	"state",
	"desc",
}

func (c *addCachingService) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addCachingService) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	var cachingServices []*ufspb.CachingService
	if c.newSpecsFile != "" {
		if utils.IsCSVFile(c.newSpecsFile) {
			cachingServices, err = c.parseMCSV()
			if err != nil {
				return err
			}
		} else {
			if err = utils.ParseJSONFile(c.newSpecsFile, &cs); err != nil {
				return err
			}
		}
	} else {
		c.parseArgs(&cs)
	}
	if len(cachingServices) == 0 {
		cachingServices = append(cachingServices, &cs)
	}
	for _, r := range cachingServices {
		res, err := ic.CreateCachingService(ctx, &ufsAPI.CreateCachingServiceRequest{
			CachingService:   r,
			CachingServiceId: r.GetName(),
		})
		if err != nil {
			fmt.Printf("Failed to add CachingService %s. %s\n", r.GetName(), err)
			continue
		}
		res.Name = ufsUtil.RemovePrefix(res.Name)
		utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
		fmt.Printf("Successfully added the CachingService %s\n", res.Name)
	}
	return nil
}

func (c *addCachingService) parseArgs(cs *ufspb.CachingService) {
	cs.Name = c.name
	cs.Port = int32(c.port)
	cs.ServingSubnets = c.subnets
	cs.PrimaryNode = c.primary
	cs.SecondaryNode = c.secondary
	cs.State = ufsUtil.ToUFSState(c.state)
	cs.Description = c.description
}

func (c *addCachingService) validateArgs() error {
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
		if !ufsAPI.HostnameRegex.MatchString(c.name) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' must be a hostname or ipv4 address.")
		}
		if len(c.subnets) == 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-subnets' is required.")
		}
		if !ufsAPI.HostnameRegex.MatchString(c.primary) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-primary' must be a hostname or ipv4 address.")
		}
		if !ufsAPI.HostnameRegex.MatchString(c.secondary) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-secondary' must be a hostname or ipv4 address.")
		}
		if c.state != "" && !ufsUtil.IsUFSState(ufsUtil.RemoveStatePrefix(c.state)) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid state name, please check help info for '-state'.", c.state)
		}
	}
	return nil
}

// parseMCSV parses the MCSV file and returns CachingService requests.
func (c *addCachingService) parseMCSV() ([]*ufspb.CachingService, error) {
	records, err := utils.ParseMCSVFile(c.newSpecsFile)
	if err != nil {
		return nil, err
	}
	var cachingServices []*ufspb.CachingService
	for i, rec := range records {
		// if i is 0, determine whether this is a header.
		if i == 0 && utils.LooksLikeHeader(rec) {
			if err := utils.ValidateSameStringArray(mcsvFields, rec); err != nil {
				return nil, err
			}
			continue
		}
		cs := &ufspb.CachingService{}
		for i := range mcsvFields {
			name := mcsvFields[i]
			value := rec[i]
			switch name {
			case "name":
				if !ufsAPI.HostnameRegex.MatchString(value) {
					return nil, fmt.Errorf("Error in line %d.\nFailed to parse name(must be a hostname or ipv4 address) %s", i, value)
				}
				cs.Name = value
			case "port":
				port, err := strconv.ParseInt(value, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("Error in line %d.\nFailed to parse port %s", i, value)
				}
				cs.Port = int32(port)
			case "subnets":
				cs.ServingSubnets = strings.Fields(value)
			case "primary":
				if !ufsAPI.HostnameRegex.MatchString(value) {
					return nil, fmt.Errorf("Error in line %d.\nFailed to parse primary(must be a hostname or ipv4 address) %s", i, value)
				}
				cs.PrimaryNode = value
			case "secondary":
				if !ufsAPI.HostnameRegex.MatchString(value) {
					return nil, fmt.Errorf("Error in line %d.\nFailed to parse secondary(must be a hostname or ipv4 address) %s", i, value)
				}
				cs.SecondaryNode = value
			case "state":
				if !ufsUtil.IsUFSState(ufsUtil.RemoveStatePrefix(value)) {
					return nil, fmt.Errorf("Error in line %d.\n%s is not a valid state name. %s", i, value, cmdhelp.StateFilterHelpText)
				}
				cs.State = ufsUtil.ToUFSState(value)
			case "desc":
				cs.Description = value
			default:
				return nil, fmt.Errorf("Error in line %d.\nUnknown field: %s", i, name)
			}
		}
		cachingServices = append(cachingServices, cs)
	}
	return cachingServices, nil
}
