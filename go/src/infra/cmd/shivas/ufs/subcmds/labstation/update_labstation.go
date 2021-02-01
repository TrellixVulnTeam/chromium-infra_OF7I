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
	"google.golang.org/genproto/protobuf/field_mask"

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

const (
	// LSE related UpdateMask paths.
	machinesPath    = "machines"
	descriptionPath = "description"
	tagsPath        = "tags"
	ticketPath      = "deploymentTicket"

	// RPM related UpdateMask paths.
	rpmHostPath   = "labstation.rpm.host"
	rpmOutletPath = "labstation.rpm.outlet"

	// Labstation related UpdateMask paths.
	poolsPath = "labstation.pools"
)

// UpdateLabstationCmd update dut by given hostname and start a swarming job to deploy.
var UpdateLabstationCmd = &subcommands.Command{
	UsageLine: "labstation [options]",
	ShortDesc: "Update a labstation",
	LongDesc:  cmdhelp.UpdateLabstationLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateLabstation{
			pools:         []string{},
			deployTags:    shivasTags,
			deployActions: defaultDeployTaskActions,
		}
		// Initialize servo setup types.
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.LabstationUpdateFileText)

		c.Flags.StringVar(&c.hostname, "name", "", "hostname of the Labstation.")
		c.Flags.StringVar(&c.machine, "asset", "", "asset tag of the Labstation.")
		c.Flags.Var(utils.CSVString(&c.pools), "pools", "comma seperated pools assigned to the Labstation.")
		c.Flags.StringVar(&c.rpm, "rpm", "", "rpm assigned to the Labstation. Clearing this field will delete rpm. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.rpmOutlet, "rpm-outlet", "", "rpm outlet used for the Labstation.")
		c.Flags.StringVar(&c.deploymentTicket, "ticket", "", "the deployment ticket for this machine. "+cmdhelp.ClearFieldHelpText)
		c.Flags.Var(utils.CSVString(&c.tags), "tags", "comma separated tags. You can only append new tags or delete all of them. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.description, "desc", "", "description for the machine. "+cmdhelp.ClearFieldHelpText)

		c.Flags.Int64Var(&c.deployTaskTimeout, "deploy-timeout", swarming.DeployTaskExecutionTimeout, "execution timeout for deploy task in seconds.")
		c.Flags.BoolVar(&c.forceDeploy, "force-deploy", false, "forces a redeploy task.")
		c.Flags.Var(utils.CSVString(&c.deployTags), "deploy-tags", "comma seperated tags for deployment task.")
		return c
	},
}

type updateLabstation struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	// Labstation specification inputs.
	newSpecsFile     string
	hostname         string
	machine          string
	pools            []string
	rpm              string
	rpmOutlet        string
	deploymentTicket string
	tags             []string
	description      string

	// Deploy task inputs.
	forceDeploy       bool
	deployTaskTimeout int64
	deployActions     []string
	deployTags        []string
}

func (c *updateLabstation) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateLabstation) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	// Using a map to collect deploy tasks. This ensures single deploy task per Labstation.
	var deployTasks map[string]*ufsAPI.UpdateMachineLSERequest

	// Create a summary results table with 3 columns.
	resTable := utils.NewSummaryResultsTable([]string{"Labstation", ufsOp, swarmingOp})

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
	c.verbosePrint("Using UFS service %s \n", e.UnifiedFleetService)
	c.verbosePrint("Using swarming service %s \n", e.SwarmingService)

	requests, err := c.parseArgs()
	if err != nil {
		return err
	}

	deployTasks = make(map[string]*ufsAPI.UpdateMachineLSERequest)
	for _, req := range requests {
		// Collect all the duts into a map.
		deployTasks[req.MachineLSE.GetName()] = req
	}

	// Update the UFS database.
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})

	for _, req := range requests {

		err := c.updateLabstationToUFS(ctx, ic, req)
		resTable.RecordResult(ufsOp, req.MachineLSE.GetHostname(), err)
		if err != nil {
			if !c.forceDeploy {
				c.verbosePrint("[%s] Unable to update, Skipping deployment: %s\n", req.MachineLSE.GetHostname(), err.Error())
				// Skip deploy task if there was an error updating to UFS.
				delete(deployTasks, req.MachineLSE.GetHostname())
				// Record skipping deploy task
				resTable.RecordSkip(swarmingOp, req.MachineLSE.GetHostname(), err.Error())
			} else {
				c.verbosePrint("[%s] Unable to update: %s\n", req.MachineLSE.GetHostname(), err.Error())
			}
		}
	}

	// Check and start deploy tasks for required Labstations.
	if len(deployTasks) > 0 {
		//TODO(anushruth): Change to e.SwarmingService once UFS migration is complete.
		tc, err := swarming.NewTaskCreator(ctx, &c.authFlags, "https://chromium-swarm-dev.appspot.com/")
		if err != nil {
			return err
		}
		tc.LogdogService = e.LogdogService
		//TODO(anushruth): Change to e.SwarmingServiceAccount once UFS migration is complete.
		tc.SwarmingServiceAccount = "skylab-admin-task@chromeos-service-accounts-dev.iam.gserviceaccount.com"

		for _, req := range deployTasks {
			// Check if deploy task is required or force deploy is set.
			if c.forceDeploy || c.isDeployTaskRequired(req) {
				err := c.deployLabstationToSwarming(ctx, tc, req.MachineLSE)
				if err != nil {
					c.verbosePrint("Unable to deploy task for %s: %s\n", req.MachineLSE.GetHostname(), err.Error())
				}
				resTable.RecordResult(swarmingOp, req.MachineLSE.GetHostname(), err)
			} else {
				resTable.RecordSkip(swarmingOp, req.MachineLSE.GetHostname(), "Deploy task not required")
			}
		}
		// Display URL for all tasks if at least one task is triggered.
		if resTable.IsSuccessForAny(swarmingOp) {
			fmt.Printf("\nTriggered deployment task(s). Follow at: %s\n\n", tc.SessionTasksURL())
		}
	}

	fmt.Print("\nSummary of results:\n\n")
	resTable.PrintResultsTable(os.Stdout, true)

	return nil
}

// validateArgs validates the set of inputs to updateLabstation.
func (c *updateLabstation) validateArgs() error {
	if c.newSpecsFile == "" && c.hostname == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Need hostname to create a Labstation")
	}
	if c.newSpecsFile != "" {
		// Cannot accept cmdline inputs for Labstation when csv/json mode is specified.
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
		if len(c.pools) != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe MCSV/JSON mode is specified. '-pools' cannot be specified at the same time.")
		}
	}
	// If hostname is given and it's not forceDeploy. Check if no other input is given.
	if c.hostname != "" && !c.forceDeploy {
		if c.machine == "" && c.rpm == "" && c.rpmOutlet == "" && c.description == "" && c.deploymentTicket == "" && len(c.tags) == 0 && len(c.pools) == 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNothing to update")
		}
	}
	return nil
}

// isDeployTaskRequired checks if the deploy task is required for the given request.
func (c *updateLabstation) isDeployTaskRequired(req *ufsAPI.UpdateMachineLSERequest) bool {
	if req.UpdateMask == nil || len(req.UpdateMask.Paths) == 0 {
		// Cannot skip deploy task. Generating a full update.
		return true
	}
	// If machine is being updated. Then we cannot skip the deploy task.
	if containsAnyStrings(req.UpdateMask.Paths, machinesPath) {
		return true
	}
	// If rpm is being updated. Then we cannot skip the deploy task.
	if containsAnyStrings(req.UpdateMask.Paths, rpmHostPath, rpmOutletPath) {
		return true
	}
	return false
}

// validateRequest checks if the req is valid based on the cmdline input.
func (c *updateLabstation) validateRequest(ctx context.Context, req *ufsAPI.UpdateMachineLSERequest) error {
	lse := req.MachineLSE
	mask := req.UpdateMask
	if mask == nil || len(mask.Paths) == 0 {
		if lse == nil {
			return fmt.Errorf("Internal Error. Invalid UpdateMachineLSERequest")
		}
		if lse.Name == "" {
			return fmt.Errorf("Invalid update. Missing Labstation name")
		}
	}
	return nil
}

// containsAnyStrings returns true if any of the inputs are included in the list.
func containsAnyStrings(list []string, inputs ...string) bool {
	for _, a := range list {
		for _, b := range inputs {
			if b == a {
				return true
			}
		}
	}
	return false
}

// parseArgs reads input from the cmd line parameters and generates update dut request.
func (c *updateLabstation) parseArgs() ([]*ufsAPI.UpdateMachineLSERequest, error) {
	if c.newSpecsFile != "" {
		if utils.IsCSVFile(c.newSpecsFile) {
			return c.parseMCSV()
		}
		machineLse := &ufspb.MachineLSE{}
		if err := utils.ParseJSONFile(c.newSpecsFile, machineLse); err != nil {
			return nil, err
		}
		// json input updates without a mask.
		return []*ufsAPI.UpdateMachineLSERequest{{
			MachineLSE: machineLse,
		}}, nil
	}

	lse, mask, err := c.initializeLSEAndMask(nil)
	if err != nil {
		return nil, err
	}
	return []*ufsAPI.UpdateMachineLSERequest{{
		MachineLSE: lse,
		UpdateMask: mask,
	}}, nil
}

// parseMCSV generates update request from mcsv file.
func (c *updateLabstation) parseMCSV() ([]*ufsAPI.UpdateMachineLSERequest, error) {
	records, err := utils.ParseMCSVFile(c.newSpecsFile)
	if err != nil {
		return nil, err
	}
	var requests []*ufsAPI.UpdateMachineLSERequest
	for i, rec := range records {
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
		lse, mask, err := c.initializeLSEAndMask(recMap)
		if err != nil {
			// Print the error and the line number and continue to next one.
			fmt.Printf("Error [%s:%v]: %s\n", c.newSpecsFile, i+1, err.Error())
			continue
		}
		requests = append(requests, &ufsAPI.UpdateMachineLSERequest{
			MachineLSE: lse,
			UpdateMask: mask,
		})
	}
	return requests, nil
}

func (c *updateLabstation) initializeLSEAndMask(recMap map[string]string) (*ufspb.MachineLSE, *field_mask.FieldMask, error) {
	var name, rpmHost, rpmOutlet string
	var pools, machines []string
	if recMap != nil {
		// CSV map. Assign all the params to the variables.
		name = recMap["name"]
		rpmHost = recMap["rpm_host"]
		rpmOutlet = recMap["rpm_outlet"]
		machines = []string{recMap["asset"]}
		pools = strings.Fields(recMap["pools"])
	} else {
		// command line parameters. Update vars with the correct values.
		name = c.hostname
		rpmHost = c.rpm
		rpmOutlet = c.rpmOutlet
		machines = []string{c.machine}
		pools = c.pools
	}

	// Generate lse and mask
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
	mask := &field_mask.FieldMask{}
	lse.Name = name
	lse.Hostname = name
	lse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Hostname = name

	// Check if machines are being updated.
	if len(machines) > 0 && machines[0] != "" {
		lse.Machines = machines
		mask.Paths = append(mask.Paths, machinesPath)
	}

	// Check and update pools if required.
	if len(pools) > 0 && pools[0] != "" {
		mask.Paths = append(mask.Paths, poolsPath)
		lse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Pools = pools
	}

	// Create and assign rpm and corresponding masks.
	rpm, paths := generateRPMWithMask(rpmHost, rpmOutlet)
	lse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Rpm = rpm
	mask.Paths = append(mask.Paths, paths...)

	// Check if description field is being updated/cleared.
	if c.description != "" {
		mask.Paths = append(mask.Paths, descriptionPath)
		if c.description != utils.ClearFieldValue {
			lse.Description = c.description
		} else {
			lse.Description = ""
		}
	}

	// Check if deployment ticket is being updated/cleared.
	if c.deploymentTicket != "" {
		mask.Paths = append(mask.Paths, ticketPath)
		if c.deploymentTicket != utils.ClearFieldValue {
			lse.DeploymentTicket = c.deploymentTicket
		} else {
			lse.DeploymentTicket = ""
		}
	}

	// Check if tags are being appended/deleted. Tags can either be appended or cleared.
	if len(c.tags) > 0 {
		mask.Paths = append(mask.Paths, tagsPath)
		lse.Tags = c.tags
		// Check if utils.ClearFieldValue is included in any of the tags.
		if containsAnyStrings(c.tags, utils.ClearFieldValue) {
			lse.Tags = nil
		}
	}

	// Check if nothing is being updated. Updating with an empty mask overwrites everything.
	if !c.forceDeploy && (len(mask.Paths) == 0 || mask.Paths[0] == "") {
		return nil, nil, cmdlib.NewQuietUsageError(c.Flags, "Nothing to update")
	}
	return lse, mask, nil
}

// generateRPMWithMask generates a rpm object from the given inputs and corresponding mask.
func generateRPMWithMask(rpmHost, rpmOutlet string) (*chromeosLab.OSRPM, []string) {
	// Check if rpm is being deleted.
	if rpmHost == utils.ClearFieldValue {
		// Generate mask and empty rpm.
		return nil, []string{rpmHostPath}
	}

	rpm := &chromeosLab.OSRPM{}
	paths := []string{}
	// Check and update rpm.
	if rpmHost != "" {
		rpm.PowerunitName = rpmHost
		paths = append(paths, rpmHostPath)
	}
	if rpmOutlet != "" {
		rpm.PowerunitOutlet = rpmOutlet
		paths = append(paths, rpmOutletPath)
	}
	return rpm, paths
}

// updateLabstationToUFS verifies the request and calls UpdateMachineLSE API with the given request.
func (c *updateLabstation) updateLabstationToUFS(ctx context.Context, ic ufsAPI.FleetClient, req *ufsAPI.UpdateMachineLSERequest) error {
	// Validate the update request.
	if err := c.validateRequest(ctx, req); err != nil {
		return err
	}
	// Print existing LSE before update.
	if err := utils.PrintExistingHost(ctx, ic, req.MachineLSE.GetName()); err != nil {
		return err
	}
	req.MachineLSE.Name = ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, req.MachineLSE.Name)

	res, err := ic.UpdateMachineLSE(ctx, req)
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	c.verbosePrint("Successfully updated Labstation to UFS: %s \n", res.GetName())
	return nil
}

// deployLabstationToSwarming starts a re-deploy task for the given Labstation.
func (c *updateLabstation) deployLabstationToSwarming(ctx context.Context, tc *swarming.TaskCreator, lse *ufspb.MachineLSE) error {
	var hostname, machine string
	// Using hostname because name has resource prefix
	hostname = lse.GetHostname()
	machines := lse.GetMachines()
	if len(machines) > 0 {
		machine = machines[0]
	}
	task, err := tc.DeployDut(ctx, hostname, machine, defaultSwarmingPool, c.deployTaskTimeout, c.deployActions, c.deployTags, nil)
	if err != nil {
		return err
	}
	c.verbosePrint("Triggered Deploy task for Labstation %s. Follow the deploy job at %s\n", hostname, task.TaskURL)

	return nil
}

func (c *updateLabstation) verbosePrint(format string, a ...interface{}) (int, error) {
	if c.commonFlags.Verbose() {
		return fmt.Printf(format, a...)
	}
	return 0, nil
}
