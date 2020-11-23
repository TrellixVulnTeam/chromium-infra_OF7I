// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package switches

import (
	"fmt"
	"strconv"
	"strings"

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

// AddSwitchCmd add Switch in the lab.
var AddSwitchCmd = &subcommands.Command{
	UsageLine: "switch [Options...]",
	ShortDesc: "Add a switch to a rack",
	LongDesc:  cmdhelp.AddSwitchLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addSwitch{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.SwitchFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.rackName, "rack", "", "name of the rack to associate the switch")
		c.Flags.StringVar(&c.switchName, "name", "", "the name of the switch to add")
		c.Flags.StringVar(&c.description, "desc", "", "the description of the switch to add")
		c.Flags.IntVar(&c.capacity, "capacity", 0, "indicate how many ports this switch support")
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags. You can only append/add new tags here.")
		return c
	},
}

type addSwitch struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string
	interactive  bool

	rackName    string
	switchName  string
	description string
	capacity    int
	tags        string
}

var mcsvFields = []string{
	"name",
	"rack",
	"capacity",
	"desc",
	"tags",
}

func (c *addSwitch) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addSwitch) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	var s ufspb.Switch
	var switches []*ufspb.Switch
	if c.interactive {
		utils.GetSwitchInteractiveInput(ctx, ic, &s)
	} else if c.newSpecsFile != "" {
		if utils.IsCSVFile(c.newSpecsFile) {
			switches, err = c.parseMCSV()
			if err != nil {
				return err
			}
		} else {
			if err = utils.ParseJSONFile(c.newSpecsFile, &s); err != nil {
				return err
			}
			if s.GetRack() == "" {
				return errors.New(fmt.Sprintf("rack field is empty in json. It is a required parameter for json input."))
			}
		}
	} else {
		c.parseArgs(&s)
	}
	if len(switches) == 0 {
		switches = append(switches, &s)
	}
	for _, r := range switches {
		res, err := ic.CreateSwitch(ctx, &ufsAPI.CreateSwitchRequest{
			Switch:   r,
			SwitchId: r.GetName(),
		})
		if err != nil {
			fmt.Printf("Failed to add switch %s to rack %s. %s\n", r.GetName(), r.GetRack(), err)
			continue
		}
		if err != nil {
			return err
		}
		res.Name = ufsUtil.RemovePrefix(res.Name)
		utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
		fmt.Printf("Successfully added the switch %s to rack %s\n", res.Name, res.GetRack())
	}
	return nil
}

func (c *addSwitch) parseArgs(s *ufspb.Switch) {
	s.Name = c.switchName
	s.Rack = c.rackName
	s.Description = c.description
	s.CapacityPort = int32(c.capacity)
	s.Tags = utils.GetStringSlice(c.tags)
}

func (c *addSwitch) validateArgs() error {
	if c.newSpecsFile != "" && c.interactive {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive & file mode cannot be specified at the same time.")
	}
	if c.newSpecsFile != "" || c.interactive {
		if c.switchName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/file mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.capacity != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/file mode is specified. '-capacity' cannot be specified at the same time.")
		}
		if c.description != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/file mode is specified. '-desc' cannot be specified at the same time.")
		}
		if c.tags != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/file mode is specified. '-tags' cannot be specified at the same time.")
		}
		if c.rackName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/file mode is specified. '-rack' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.switchName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f' or '-i') is specified.")
		}
		if c.capacity == 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-capacity' is required, no mode ('-f' or '-i') is specified.")
		}
		if c.rackName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nRack name (-rack) is required.")
		}
	}
	return nil
}

// parseMCSV parses the MCSV file and returns switches
func (c *addSwitch) parseMCSV() ([]*ufspb.Switch, error) {
	records, err := utils.ParseMCSVFile(c.newSpecsFile)
	if err != nil {
		return nil, err
	}
	var switches []*ufspb.Switch
	for i, rec := range records {
		// if i is 0, determine whether this is a header
		if i == 0 && utils.LooksLikeHeader(rec) {
			if err := utils.ValidateSameStringArray(mcsvFields, rec); err != nil {
				return nil, err
			}
			continue
		}
		s := &ufspb.Switch{}
		for i := range mcsvFields {
			name := mcsvFields[i]
			value := rec[i]
			switch name {
			case "name":
				s.Name = value
			case "rack":
				s.Rack = value
			case "desc":
				s.Description = value
			case "capacity":
				capacityPort, err := strconv.ParseInt(value, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("failed to parse capacity %s", value)
				}
				s.CapacityPort = int32(capacityPort)
			case "tags":
				s.Tags = strings.Fields(value)
			default:
				return nil, fmt.Errorf("unknown field: %s", name)
			}
		}
		switches = append(switches, s)
	}
	return switches, nil
}
