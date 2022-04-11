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
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/genproto/protobuf/field_mask"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	"infra/cros/recovery/buildbucket"
	"infra/libs/skylab/common/heuristics"
	swarming "infra/libs/swarming"
	ufspb "infra/unifiedfleet/api/v1/models"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// defaultDeployTaskActions is a collection of actions for SSW to setup labstation.
var defaultDeployTaskActions = []string{"setup-labstation", "update-label", "run-pre-deploy-verification"}

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
		c.Flags.Var(flag.StringSlice(&c.tags), "tag", "Name(s) of tag(s). Can be specified multiple times.")
		c.Flags.StringVar(&c.description, "desc", "", "description for the machine.")
		c.Flags.StringVar(&c.model, "model", "", "model name of the device")
		c.Flags.StringVar(&c.board, "board", "", "board the device is based on")
		c.Flags.StringVar(&c.rack, "rack", "", "rack that the labstation is on")
		c.Flags.StringVar(&c.zone, "zone", "", "zone that the labstation is on. "+cmdhelp.ZoneFilterHelpText)
		c.Flags.BoolVar(&c.paris, "paris", true, "use paris flow for deployment")
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
	paris             bool

	// Asset related params
	model string
	board string
	rack  string
	zone  string
}

var mcsvFields = []string{
	"name",
	"asset",
	"model",
	"board",
	"rpm_host",
	"rpm_outlet",
	"pools",
}

// labstationDeployUFSParams contains all the data that are needed for deployment of a single labstation
// Asset and its update paths are required here to update location, model and board for the labstation
// See: crbug.com/1188488 for why model and board need to be updated.
type labstationDeployUFSParams struct {
	Labstation *ufspb.MachineLSE // MachineLSE of the labstation to be updated
	Asset      *ufspb.Asset      // Asset underlying the labstation being updated
	Paths      []string          // Update paths for the Asset being updated
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
	ctx = utils.SetupContext(ctx, ufsUtil.OSNamespace)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}

	e := c.envFlags.Env()
	if c.commonFlags.Verbose() {
		fmt.Printf("Using UFS service %s \n", e.UnifiedFleetService)
		fmt.Printf("Using swarming service %s \n", e.SwarmingService)
	}

	tc, err := swarming.NewTaskCreator(ctx, &c.authFlags, e.SwarmingService)
	if err != nil {
		return err
	}
	tc.LogdogService = e.LogdogService
	tc.SwarmingServiceAccount = e.SwarmingServiceAccount
	deployParams, err := c.parseArgs()
	if err != nil {
		return err
	}

	resTable := utils.NewSummaryResultsTable([]string{"Labstation", ufsOp, swarmingOp})

	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})

	var bbClient buildbucket.Client
	if c.paris {
		var cErr error
		bbClient, cErr = createBBClient(ctx, c.authFlags)
		if cErr != nil {
			return cErr
		}
	}

	for _, params := range deployParams {
		if len(params.Labstation.GetMachines()) == 0 {
			fmt.Printf("Failed to add Labstation %s to UFS. It is not linked to any Asset(Machine).\n", params.Labstation.GetName())
			continue
		}
		err := c.addLabstationToUFS(ctx, ic, params)
		resTable.RecordResult(ufsOp, params.Labstation.GetHostname(), err)
		if err == nil {
			// Deploy and record result.
			dErr := c.createLabstationDeployTask(ctx, tc, bbClient, params.Labstation, e, params.Labstation.GetHostname())
			resTable.RecordResult(swarmingOp, params.Labstation.GetHostname(), dErr)
		} else {
			// Record deploy task skip.
			resTable.RecordSkip(swarmingOp, params.Labstation.GetHostname(), "")
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
		if utils.IsCSVFile(c.newSpecsFile) {
			if c.board != "" {
				return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe MCSV mode is specified. '-board' cannot be specified at the same time.")
			}
			if c.model != "" {
				return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe MCSV mode is specified. '-model' cannot be specified at the same time.")
			}
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
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe MCSV/JSON mode is specified. '-tag' cannot be specified at the same time.")
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
func (c *addLabstation) parseArgs() ([]*labstationDeployUFSParams, error) {
	if c.newSpecsFile != "" {
		if utils.IsCSVFile(c.newSpecsFile) {
			return c.parseMCSV()
		}
		machinelse := &ufspb.MachineLSE{}
		if err := utils.ParseJSONFile(c.newSpecsFile, machinelse); err != nil {
			return nil, err
		}
		machinelse.Hostname = machinelse.Name
		if len(machinelse.GetMachines()) == 0 {
			return nil, cmdlib.NewQuietUsageError(c.Flags, "Need asset tag to create Labstation. Use Machines field in %s", c.newSpecsFile)
		}
		asset, paths := utils.GenerateAssetUpdate(machinelse.GetMachines()[0], c.model, c.board, c.zone, c.rack)
		return []*labstationDeployUFSParams{{
			Labstation: machinelse,
			Asset:      asset,
			Paths:      paths,
		}}, nil
	}
	// command line parameters
	deployParams, err := c.initializeLSEAndAsset(nil)
	if err != nil {
		return nil, err
	}
	return []*labstationDeployUFSParams{deployParams}, nil
}

// parseMCSV parses the MCSV file and returns MachineLSEs
func (c *addLabstation) parseMCSV() ([]*labstationDeployUFSParams, error) {
	records, err := utils.ParseMCSVFile(c.newSpecsFile)
	if err != nil {
		return nil, err
	}
	var deployParams []*labstationDeployUFSParams
	for i, rec := range records {
		// if i is 1, determine whether this is a header
		if i == 0 && heuristics.LooksLikeHeader(rec) {
			if err := utils.ValidateSameStringArray(mcsvFields, rec); err != nil {
				return nil, err
			}
			continue
		}
		recMap := make(map[string]string)
		for j, title := range mcsvFields {
			recMap[title] = rec[j]
		}
		params, err := c.initializeLSEAndAsset(recMap)
		if err != nil {
			return nil, err
		}
		deployParams = append(deployParams, params)
	}
	return deployParams, nil
}

// addLabstationToUFS attempts to create a machineLSE object in UFS.
func (c *addLabstation) addLabstationToUFS(ctx context.Context, ic ufsAPI.FleetClient, params *labstationDeployUFSParams) error {
	if params.Asset != nil {
		if err := c.updateAssetToUFS(ctx, ic, params.Asset, params.Paths); err != nil {
			return err
		}
	}
	utils.PrintProtoJSON(params.Labstation, true)
	res, err := ic.CreateMachineLSE(ctx, &ufsAPI.CreateMachineLSERequest{
		MachineLSE:   params.Labstation,
		MachineLSEId: params.Labstation.GetName(),
	})
	if err != nil {
		fmt.Printf("Failed to add Labstation %s to UFS. UFS add failed %s\n", params.Labstation.GetName(), err)
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	return nil
}

// CreateLabstationDeployTask creates a task using either the paris or legacy flow to deploy a labstation.
func (c *addLabstation) createLabstationDeployTask(ctx context.Context, tc *swarming.TaskCreator, bbClient buildbucket.Client, lse *ufspb.MachineLSE, e site.Environment, host string) error {
	if c.paris {
		return utils.ScheduleDeployTask(ctx, bbClient, e, host)
	} else {
		task, dErr := tc.DeployDut(ctx, lse.Name, lse.GetMachines()[0], defaultSwarmingPool, c.deployTaskTimeout, c.deployActions, c.deployTags, nil)
		if dErr != nil {
			return errors.Annotate(dErr, "deploy labstation").Err()
		}
		fmt.Printf("Triggered Deploy task for Labstation %s. Follow the deploy job at %s\n", lse.GetName(), task.TaskURL)
	}
	return nil
}

// CreateBBClient creates a buildbucket client if permitted.
func createBBClient(ctx context.Context, authFlags authcli.Flags) (buildbucket.Client, error) {
	bc, err := buildbucket.NewLabpackClient(ctx, authFlags, site.DefaultPRPCOptions)
	if err != nil {
		return nil, errors.Annotate(err, "ensure bb client").Err()
	}
	return bc, nil
}

func (c *addLabstation) initializeLSEAndAsset(recMap map[string]string) (*labstationDeployUFSParams, error) {
	lse := &ufspb.MachineLSE{
		Lse: &ufspb.MachineLSE_ChromeosMachineLse{
			ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
				ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
					DeviceLse: &ufspb.ChromeOSDeviceLSE{
						Device: &ufspb.ChromeOSDeviceLSE_Labstation{
							Labstation: &chromeosLab.Labstation{
								Rpm: &chromeosLab.OSRPM{},
							},
						},
					},
				},
			},
		},
	}
	var name, rpmHost, rpmOutlet, model, board string
	var asset *ufspb.Asset
	var pools, machines, paths []string
	if recMap != nil {
		// CSV map
		name = recMap["name"]
		rpmHost = recMap["rpm_host"]
		rpmOutlet = recMap["rpm_outlet"]
		model = recMap["model"]
		board = recMap["board"]
		machines = []string{recMap["asset"]}
		pools = strings.Fields(recMap["pools"])
	} else {
		// command line parameters
		name = c.hostname
		rpmHost = c.rpm
		rpmOutlet = c.rpmOutlet
		model = c.model
		board = c.board
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
	// Get the updated asset and update paths
	// Note: If it's a csv update, allow for updating zone and rack from cmdline
	asset, paths = utils.GenerateAssetUpdate(machines[0], model, board, c.zone, c.rack)
	return &labstationDeployUFSParams{
		Labstation: lse,
		Asset:      asset,
		Paths:      paths,
	}, nil
}

// updateAssetToUFS calls UpdateAsset API in UFS with asset and partial paths
func (c *addLabstation) updateAssetToUFS(ctx context.Context, ic ufsAPI.FleetClient, asset *ufspb.Asset, paths []string) error {
	if len(paths) == 0 {
		// If no update is available. Skip doing anything
		return nil
	}
	mask := &field_mask.FieldMask{
		Paths: paths,
	}
	_, err := ic.UpdateAsset(ctx, &ufsAPI.UpdateAssetRequest{
		Asset:      asset,
		UpdateMask: mask,
	})
	return err
}
