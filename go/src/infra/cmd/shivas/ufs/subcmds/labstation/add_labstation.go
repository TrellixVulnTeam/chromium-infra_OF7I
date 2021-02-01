// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package labstation

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	swarming "infra/libs/swarming"
	ufspb "infra/unifiedfleet/api/v1/models"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// defaultDeployTaskActions is a collection of actions for SSW to setup labstation.
var defaultDeployTaskActions = []string{"setup-labstation", "update-label"}

var shivasTags = []string{"shivas:" + site.VersionNumber, "triggered_using:shivas"}

// defaultPools contains the list of  pools used by default.
var defaultPools = []string{"labstation_main"}

// defaultSwarmingPool is the swarming pool used for all Labstations.
var defaultSwarmingPool = "ChromeOSSkylab"

var ufsOp = "Update to database" // Summary table column for update operation.
var swarmingOp = "Deploy task"   // Summary table column for deploy operation.

// AddLabstationCmd adds a MachineLSE to the database. And starts a swarming job to deploy.
var AddLabstationCmd = &subcommands.Command{
	UsageLine: "labstation [options ...]",
	ShortDesc: "Deploy a labstation",
	LongDesc:  cmdhelp.AddLabstationLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addLabstation{
			pools:         []string{},
			deployTags:    shivasTags,
			deployActions: defaultDeployTaskActions,
		}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.LabstationRegistrationFileText)

		c.Flags.StringVar(&c.hostname, "name", "", "hostname of the Labstation.")
		c.Flags.StringVar(&c.machine, "asset", "", "asset tag of the Labstation.")
		c.Flags.Var(utils.CSVString(&c.pools), "pools", "comma seperated pools assigned to the Labstation. 'labstation_main' assigned on no input.")
		c.Flags.StringVar(&c.rpm, "rpm", "", "rpm assigned to the Labstation.")
		c.Flags.StringVar(&c.rpmOutlet, "rpm-outlet", "", "rpm outlet used for the Labstation.")
		c.Flags.Int64Var(&c.deployTaskTimeout, "deploy-timeout", swarming.DeployTaskExecutionTimeout, "execution timeout for deploy task in seconds.")
		c.Flags.Var(utils.CSVString(&c.deployTags), "deploy-tags", "comma seperated tags for deployment task.")
		c.Flags.StringVar(&c.deploymentTicket, "ticket", "", "the deployment ticket for this machine.")
		c.Flags.Var(utils.CSVString(&c.tags), "tags", "comma separated tags for the Labstation.")
		c.Flags.StringVar(&c.description, "desc", "", "description for the machine.")
		return c
	},
}

type addLabstation struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string
	hostname     string
	machine      string
	pools        []string
	rpm          string
	rpmOutlet    string

	deployTaskTimeout int64
	deployActions     []string
	deployTags        []string
	deploymentTicket  string
	tags              []string
	state             string
	description       string
}

var mcsvFields = []string{
	"name",
	"asset",
	"rpm_host",
	"rpm_outlet",
	"pools",
}

func (c *addLabstation) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addLabstation) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
		fmt.Printf("Using UFS service %s \n", e.UnifiedFleetService)
		fmt.Printf("Using swarming service %s \n", e.SwarmingService)
	}

	//TODO(anushruth): Change to e.SwarmingService once UFS migration is complete.
	tc, err := swarming.NewTaskCreator(ctx, &c.authFlags, "https://chromium-swarm-dev.appspot.com/")
	if err != nil {
		return err
	}
	tc.LogdogService = e.LogdogService
	//TODO(anushruth): Change to e.SwarmingServiceAccount once UFS migration is complete.
	tc.SwarmingServiceAccount = "skylab-admin-task@chromeos-service-accounts-dev.iam.gserviceaccount.com"
	machineLSEs, err := c.parseArgs()
	if err != nil {
		return err
	}

	resTable := utils.NewSummaryResultsTable([]string{"Labstation", ufsOp, swarmingOp})

	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})

	for _, lse := range machineLSEs {
		if len(lse.GetMachines()) == 0 {
			fmt.Printf("Failed to add Labstation %s to UFS. It is not linked to any Asset(Machine).\n", lse.GetName())
			continue
		}
		err := c.addLabstationToUFS(ctx, ic, lse)
		resTable.RecordResult(ufsOp, lse.GetHostname(), err)
		if err == nil {
			// Deploy and record result.
			resTable.RecordResult(swarmingOp, lse.GetHostname(), c.deployLabstationToSwarming(ctx, tc, lse))
		} else {
			// Record deploy task skip.
			resTable.RecordSkip(swarmingOp, lse.GetHostname(), "")
		}
	}
	// Print session URL if atleast one of the tasks was deployed.
	if resTable.IsSuccessForAny(swarmingOp) {
		fmt.Fprintf(a.GetOut(), "\nBatch tasks URL: %s\n\n", tc.SessionTasksURL())
	}

	fmt.Println("\nSummary of operations:")
	resTable.PrintResultsTable(os.Stdout, true)

	return nil
}

// validateArgs validates the input flags.
func (c addLabstation) validateArgs() error {
	if c.newSpecsFile != "" {
		// Using file input. Cmdline inputs are not allowed.
		if c.hostname != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe MCSV/JSON mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.machine != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe MCSV/JSON mode is specified. '-asset' cannot be specified at the same time.")
		}
		if c.rpm != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe MCSV/JSON mode is specified. '-rpm' cannot be specified at the same time.")
		}
		if c.rpmOutlet != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe MCSV/JSON mode is specified. '-rpm-outlet' cannot be specified at the same time.")
		}
		if len(c.pools) > 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe MCSV/JSON mode is specified. '-pools' cannot be specified at the same time.")
		}
		if c.deploymentTicket != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe MCSV/JSON mode is specified. '-ticket' cannot be specified at the same time.")
		}
		if c.description != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe MCSV/JSON mode is specified. '-desc' cannot be specified at the same time.")
		}
		if len(c.tags) > 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe MCSV/JSON mode is specified. '-tags' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile == "" {
		if c.hostname == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Need hostname to create a DUT")
		}
		if c.machine == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Need machine ID to create a Labstation")
		}
		if (c.rpm != "" && c.rpmOutlet == "") || (c.rpm == "" && c.rpmOutlet != "") {
			return cmdlib.NewQuietUsageError(c.Flags, "Need both rpm and its outlet. [%s]-[%s] is invalid", c.rpm, c.rpmOutlet)
		}
	}
	return nil
}

// parseArgs reads the input and generates machineLSE.
func (c *addLabstation) parseArgs() ([]*ufspb.MachineLSE, error) {
	if c.newSpecsFile != "" {
		if utils.IsCSVFile(c.newSpecsFile) {
			return c.parseMCSV()
		}
		machinelse := &ufspb.MachineLSE{}
		if err := utils.ParseJSONFile(c.newSpecsFile, machinelse); err != nil {
			return nil, err
		}
		machinelse.Hostname = machinelse.Name
		return []*ufspb.MachineLSE{machinelse}, nil
	}
	// command line parameters
	lse, err := c.initializeLSE(nil)
	if err != nil {
		return nil, err
	}
	return []*ufspb.MachineLSE{lse}, nil
}

// parseMCSV parses the MCSV file and returns MachineLSEs
func (c *addLabstation) parseMCSV() ([]*ufspb.MachineLSE, error) {
	records, err := utils.ParseMCSVFile(c.newSpecsFile)
	if err != nil {
		return nil, err
	}
	var lses []*ufspb.MachineLSE
	for i, rec := range records {
		// if i is 1, determine whether this is a header
		if i == 0 && utils.LooksLikeHeader(rec) {
			if err := utils.ValidateSameStringArray(mcsvFields, rec); err != nil {
				return nil, err
			}
			continue
		}
		recMap := make(map[string]string)
		for j, title := range mcsvFields {
			recMap[title] = rec[j]
		}
		lse, err := c.initializeLSE(recMap)
		if err != nil {
			return nil, err
		}
		lses = append(lses, lse)
	}
	return lses, nil
}

// addLabstationToUFS attempts to create a machineLSE object in UFS.
func (c *addLabstation) addLabstationToUFS(ctx context.Context, ic ufsAPI.FleetClient, lse *ufspb.MachineLSE) error {
	res, err := ic.CreateMachineLSE(ctx, &ufsAPI.CreateMachineLSERequest{
		MachineLSE:   lse,
		MachineLSEId: lse.GetName(),
	})
	if err != nil {
		fmt.Printf("Failed to add Labstation %s to UFS. UFS add failed %s\n", lse.GetName(), err)
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	return nil
}

// deployLabstationToSwarming starts a deploy task for the given labstation.
func (c *addLabstation) deployLabstationToSwarming(ctx context.Context, tc *swarming.TaskCreator, lse *ufspb.MachineLSE) error {
	task, err := tc.DeployDut(ctx, lse.Name, lse.GetMachines()[0], defaultSwarmingPool, c.deployTaskTimeout, c.deployActions, c.deployTags, nil)
	if err != nil {
		return err
	}
	fmt.Printf("Triggered Deploy task for Labstation %s. Follow the deploy job at %s\n", lse.GetName(), task.TaskURL)
	return nil
}

func (c *addLabstation) initializeLSE(recMap map[string]string) (*ufspb.MachineLSE, error) {
	lse := &ufspb.MachineLSE{
		Lse: &ufspb.MachineLSE_ChromeosMachineLse{
			ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
				ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
					DeviceLse: &ufspb.ChromeOSDeviceLSE{
						Device: &ufspb.ChromeOSDeviceLSE_Labstation{
							Labstation: &chromeosLab.Labstation{
								Rpm: &chromeosLab.RPM{},
							},
						},
					},
				},
			},
		},
	}
	var name, rpmHost, rpmOutlet string
	var pools, machines []string
	if recMap != nil {
		// CSV map
		name = recMap["name"]
		rpmHost = recMap["rpm_host"]
		rpmOutlet = recMap["rpm_outlet"]
		machines = []string{recMap["machine"]}
		pools = strings.Fields(recMap["pools"])
	} else {
		// command line parameters
		name = c.hostname
		rpmHost = c.rpm
		rpmOutlet = c.rpmOutlet
		machines = []string{c.machine}
		pools = c.pools
	}

	// Check if machine is nil.
	if len(machines) == 0 || machines[0] == "" {
		return nil, fmt.Errorf("Cannot create labstation without asset")
	}

	lse.Name = name
	lse.Hostname = name
	lse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Hostname = name
	lse.Machines = machines

	// Use the input params if available for all the options.
	lse.Description = c.description
	lse.DeploymentTicket = c.deploymentTicket
	lse.Tags = c.tags

	// Check and assign rpm
	if (rpmHost != "" && rpmOutlet == "") || (rpmHost == "" && rpmOutlet != "") {
		return nil, fmt.Errorf("Need both rpm and its outlet. [%s]-[%s] is invalid", rpmHost, rpmOutlet)
	}
	lse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetRpm().PowerunitName = rpmHost
	lse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetRpm().PowerunitOutlet = rpmOutlet
	if len(pools) == 0 || pools[0] == "" {
		lse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Pools = defaultPools
	} else {
		lse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Pools = pools
	}
	return lse, nil
}
