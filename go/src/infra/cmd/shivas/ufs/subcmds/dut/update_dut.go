// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
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
	// Servo related UpdateMask paths.
	servoHostPath     = "dut.servo.hostname"
	servoPortPath     = "dut.servo.port"
	servoSerialPath   = "dut.servo.serial"
	servoSetupPath    = "dut.servo.setup"
	servoTypePath     = "dut.servo.type"
	servoTopologyPath = "dut.servo.topology"

	// LSE related UpdateMask paths.
	machinesPath    = "machines"
	descriptionPath = "description"
	tagsPath        = "tags"
	ticketPath      = "deploymentTicket"

	// RPM related UpdateMask paths.
	rpmHostPath   = "dut.rpm.host"
	rpmOutletPath = "dut.rpm.outlet"

	// DUT related UpdateMask paths.
	poolsPath = "dut.pools"

	// Operations string for Summary table.
	ufsOp   = "Update to Database"
	swarmOp = "Deployment"
)

// partialUpdateDeployPaths is a collection of paths for which there is a partial update on servo/rpm.
var partialUpdateDeployPaths = []string{servoHostPath, servoPortPath, servoSerialPath, servoSetupPath, rpmHostPath, rpmOutletPath}

// partialUpdateDeployActions is a collection of actions for the deploy task when updating servo/rpm.
var partialUpdateDeployActions = []string{
	"run-pre-deploy-verification",
}

// partialUpdateDeployActions is a collection of actions for the deploy task when updating machines.
var assetUpdateDeployActions = []string{
	"update-label",
	"verify-recovery-mode",
	"run-pre-deploy-verification",
	"stage-usb",
	"install-test-image",
	"install-firmware",
	"verify-recovery-mode",
}

// UpdateDUTCmd update dut by given hostname and start a swarming job to delpoy.
var UpdateDUTCmd = &subcommands.Command{
	UsageLine: "dut [options]",
	ShortDesc: "Update a DUT",
	LongDesc:  cmdhelp.UpdateDUTLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateDUT{
			pools:      []string{},
			deployTags: shivasTags,
		}
		// Initialize servo setup types
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.DUTUpdateFileText)

		c.Flags.StringVar(&c.hostname, "name", "", "hostname of the DUT.")
		c.Flags.StringVar(&c.machine, "asset", "", "asset tag of the DUT.")
		c.Flags.StringVar(&c.servo, "servo", "", "servo hostname and port as hostname:port. Clearing this field will delete the servo in DUT. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.servoSerial, "servo-serial", "", "serial number for the servo.")
		c.Flags.StringVar(&c.servoSetupType, "servo-setup", "", "servo setup type. Allowed values are "+cmdhelp.ServoSetupTypeAllowedValuesString()+".")
		c.Flags.Var(utils.CSVString(&c.pools), "pools", "comma seperated pools assigned to the DUT.")
		c.Flags.StringVar(&c.rpm, "rpm", "", "rpm assigned to the DUT. Clearing this field will delete rpm. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.rpmOutlet, "rpm-outlet", "", "rpm outlet used for the DUT.")
		c.Flags.StringVar(&c.deploymentTicket, "ticket", "", "the deployment ticket for this machine. "+cmdhelp.ClearFieldHelpText)
		c.Flags.Var(utils.CSVString(&c.tags), "tags", "comma separated tags. You can only append new tags or delete all of them. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.description, "desc", "", "description for the machine. "+cmdhelp.ClearFieldHelpText)

		c.Flags.Int64Var(&c.deployTaskTimeout, "deploy-timeout", swarming.DeployTaskExecutionTimeout, "execution timeout for deploy task in seconds.")
		c.Flags.BoolVar(&c.forceDeploy, "force-deploy", false, "forces a deploy task for all the updates.")
		c.Flags.Var(utils.CSVString(&c.deployTags), "deploy-tags", "comma seperated tags for deployment task.")
		c.Flags.BoolVar(&c.forceDownloadImage, "force-download-image", false, "force download image and stage usb if deploy task is run.")
		c.Flags.BoolVar(&c.forceInstallFirmware, "force-install-fw", false, "force install firmware if deploy task is run.")
		c.Flags.BoolVar(&c.forceInstallOS, "force-install-os", false, "force install os image if deploy task is run.")
		c.Flags.BoolVar(&c.forceUpdateLabels, "force-update-labels", false, "force update labels if deploy task is run.")
		return c
	},
}

type updateDUT struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	// DUT specification inputs.
	newSpecsFile     string
	hostname         string
	machine          string
	servo            string
	servoSerial      string
	servoSetupType   string
	pools            []string
	rpm              string
	rpmOutlet        string
	deploymentTicket string
	tags             []string
	description      string

	// Deploy task inputs.
	forceDeploy          bool
	deployTaskTimeout    int64
	deployTags           []string
	forceDownloadImage   bool
	forceInstallOS       bool
	forceInstallFirmware bool
	forceUpdateLabels    bool
}

func (c *updateDUT) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateDUT) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	// Using a map to collect deploy actions. This ensures single deploy task per DUT.
	var deployTasks map[string][]string

	// Create a summary results table with 3 columns.
	resTable := utils.NewSummaryResultsTable([]string{"DUT", ufsOp, swarmOp})

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

	requests, err := c.parseArgs()
	if err != nil {
		return err
	}

	// Create a map of DUTs to avoid triggering multiple tasks.
	deployTasks = make(map[string][]string)

	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})

	for _, req := range requests {

		// Collect the deploy actions required for the request. This is done before DUT is changed on UFS.
		actions, err := c.getDeployActions(ctx, ic, req)
		if err != nil {
			return err
		}

		// Attempt to update UFS.
		err = c.updateDUTToUFS(ctx, ic, req)
		// Record the result of the action.
		resTable.RecordResult(ufsOp, req.MachineLSE.GetName(), err)
		if err != nil {
			// Print err and skip deployment if it's not forced.
			if !c.forceDeploy {
				fmt.Printf("[%s] Error updating UFS. Skip triggering deploy task. %s\n", req.MachineLSE.GetName(), err.Error())
				// Record the skip result.
				resTable.RecordSkip(swarmOp, req.MachineLSE.GetName(), err.Error())
				continue
			}
			fmt.Printf("[%s] Failed to update UFS. Attempting to trigger deploy task '-force-deploy'. %s\n", req.MachineLSE.GetName(), err.Error())
		}
		deployTasks[req.MachineLSE.GetName()] = actions

	}

	//TODO(anushruth): Change to e.SwarmingService once UFS migration is complete.
	tc, err := swarming.NewTaskCreator(ctx, &c.authFlags, "https://chromium-swarm-dev.appspot.com/")
	if err != nil {
		return err
	}
	tc.LogdogService = e.LogdogService
	//TODO(anushruth): Change to e.SwarmingServiceAccount once UFS migration is complete.
	tc.SwarmingServiceAccount = "skylab-admin-task@chromeos-service-accounts-dev.iam.gserviceaccount.com"
	for _, req := range requests {
		// Check if the deployment is needed.
		actions, ok := deployTasks[req.MachineLSE.GetName()]
		if !ok {
			// Deploy Task not required.
			continue
		}
		// Swarm a deploy task if required or enforced.
		if len(actions) > 0 || c.forceDeploy {
			// If deploy task is enforced and len(actions) = 0 and use partialUpdatedeployActions as default.
			if len(actions) == 0 && c.forceDeploy {
				actions = partialUpdateDeployActions
			}

			// Include any enforced actions.
			actions = c.updateDeployActions(actions)
			// Start a swarming deploy task for the DUT.
			if err := c.deployDUTToSwarming(ctx, tc, req.GetMachineLSE(), actions); err != nil {
				// Print err and continue to trigger next one
				fmt.Printf("[%s] Failed to deploy task. %s", req.GetMachineLSE().GetName(), err.Error())
			}
			resTable.RecordResult(swarmOp, req.MachineLSE.GetName(), err)

			// Remove the task entry to avoid triggering multiple tasks.
			delete(deployTasks, req.MachineLSE.GetName())
		}
	}

	if resTable.IsSuccessForAny(swarmOp) {
		// Display URL for all tasks if there are more than one.
		fmt.Printf("\nTriggered deployment task(s). Follow at: %s\n", tc.SessionTasksURL())
	}

	fmt.Printf("\nSummary of results:\n\n")
	resTable.PrintResultsTable(os.Stdout, true)

	return nil
}

// validateArgs validates the set of inputs to updateDUT.
func (c updateDUT) validateArgs() error {
	if c.newSpecsFile == "" && c.hostname == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Need hostname to create a DUT")
	}
	if c.newSpecsFile == "" {
		// Check if servo input is valid
		if c.servo != "" && c.servo != utils.ClearFieldValue {
			_, _, err := parseServoHostnamePort(c.servo)
			if err != nil {
				return err
			}
		}
		// Check if servo type is valid.
		// Note: This check is run irrespective of servo input because it is possible to perform an update on only this field.
		if _, ok := chromeosLab.ServoSetupType_value[appendServoSetupPrefix(c.servoSetupType)]; c.servoSetupType != "" && !ok {
			return cmdlib.NewQuietUsageError(c.Flags, "Invalid value for servo setup type. Valid values are "+cmdhelp.ServoSetupTypeAllowedValuesString())
		}
	}
	if c.newSpecsFile != "" {
		// Helper function to return the formatted error.
		f := func(input string) error {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf("Wrong usage!!\nThe MCSV/JSON mode is specified. '-%s' cannot be specified at the same time.", input))
		}
		// Cannot accept cmdline inputs for DUT when csv/json mode is specified.
		if c.hostname != "" {
			return f("name")
		}
		if c.machine != "" {
			return f("asset")
		}
		if c.servo != "" {
			return f("servo")
		}
		if c.servoSerial != "" {
			return f("servo-serial")
		}
		if c.servoSetupType != "" {
			return f("servo-setup")
		}
		if c.rpm != "" {
			return f("rpm")
		}
		if c.rpmOutlet != "" {
			return f("rpm-outlet")
		}
		if len(c.pools) != 0 {
			return f("pools")
		}
	}
	return nil
}

// validateRequest checks if the req is valid based on the cmdline input.
func (c *updateDUT) validateRequest(ctx context.Context, req *ufsAPI.UpdateMachineLSERequest) error {
	lse := req.MachineLSE
	mask := req.UpdateMask
	if mask == nil || len(mask.Paths) == 0 {
		if lse == nil {
			return fmt.Errorf("Internal Error. Invalid UpdateMachineLSERequest")
		}
		if lse.Name == "" {
			return fmt.Errorf("Invalid update. Missing DUT name")
		}
	}
	return nil
}

// parseArgs reads input from the cmd line parameters and generates update dut request.
func (c *updateDUT) parseArgs() ([]*ufsAPI.UpdateMachineLSERequest, error) {
	if c.newSpecsFile != "" {
		if utils.IsCSVFile(c.newSpecsFile) {
			return c.parseMCSV()
		}
		machineLse := &ufspb.MachineLSE{}
		if err := utils.ParseJSONFile(c.newSpecsFile, machineLse); err != nil {
			return nil, err
		}
		if err := c.validateDUTFromJSON(machineLse); err != nil {
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

// validateDUTFromJSON checks if the input lse represents DUT and ensures servo/rpm isn't incomplete.
func (c *updateDUT) validateDUTFromJSON(dutLse *ufspb.MachineLSE) error {
	if err := utils.IsDUT(dutLse); err != nil {
		return errors.Annotate(err, "The LSE in %s is not a DUT", c.newSpecsFile).Err()
	}
	if servo := dutLse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo(); servo != nil {
		// Avoid incomplete servo updates.
		if !(servo.GetServoHostname() != "" && servo.GetServoSerial() != "") {
			// Note: ServoPort == int32(0) auto-assigns the port.
			return cmdlib.NewQuietUsageError(c.Flags, "Incomplete/Invalid servo update in %s", c.newSpecsFile)
		}
		// Don't allow updates to ServoType or ServoTopology from here, unless its to clear them both by setting servoType to ClearFieldValue.
		if servo.GetServoType() != "" && servo.GetServoType() != utils.ClearFieldValue {
			return cmdlib.NewQuietUsageError(c.Flags, "Cannot set servo_type to %s in %s. Setting it to '%s' will update both servoType and servoTopology with correct values", servo.GetServoType(), c.newSpecsFile, utils.ClearFieldValue)
		}
		// Don't allow updates to servoTopology
		if servo.GetServoTopology() != nil {
			return cmdlib.NewQuietUsageError(c.Flags, "Cannot update ServoTopology using %s. Invalid usage", c.newSpecsFile)
		}
	}
	if rpm := dutLse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetRpm(); rpm != nil {
		if (rpm.GetPowerunitName() != "" && rpm.GetPowerunitOutlet() == "") || (rpm.GetPowerunitName() == "" && rpm.GetPowerunitOutlet() != "") {
			return cmdlib.NewQuietUsageError(c.Flags, "Cannot update incomplete RPM. Need both host and outlet")
		}
	}
	return nil
}

// parseMCSV generates update request from mcsv file.
func (c *updateDUT) parseMCSV() ([]*ufsAPI.UpdateMachineLSERequest, error) {
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

func (c *updateDUT) initializeLSEAndMask(recMap map[string]string) (*ufspb.MachineLSE, *field_mask.FieldMask, error) {
	var name, servo, servoSerial, servoSetup, rpmHost, rpmOutlet string
	var pools, machines []string
	if recMap != nil {
		// CSV map. Assign all the params to the variables.
		name = recMap["name"]
		// Generate cmdline servo input. This allows for easier validation and assignment.
		if recMap["servo_host"] != "" || recMap["servo_port"] != "" {
			servo = fmt.Sprintf("%s:%s", recMap["servo_host"], recMap["servo_port"])
		}
		servoSerial = recMap["servo_serial"]
		if recMap["servo_setup"] != "" {
			servoSetup = appendServoSetupPrefix(recMap["servo_setup"])
		}
		rpmHost = recMap["rpm_host"]
		rpmOutlet = recMap["rpm_outlet"]
		machines = []string{recMap["asset"]}
		pools = strings.Fields(recMap["pools"])
	} else {
		// command line parameters. Update vars with the correct values.
		name = c.hostname
		servo = c.servo
		servoSerial = c.servoSerial
		if c.servoSetupType != "" {
			servoSetup = appendServoSetupPrefix(c.servoSetupType)
		}
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
						Device: &ufspb.ChromeOSDeviceLSE_Dut{
							Dut: &chromeosLab.DeviceUnderTest{
								Peripherals: &chromeosLab.Peripherals{},
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
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().Hostname = name

	// Check if machines are being updated.
	if len(machines) > 0 && machines[0] != "" {
		lse.Machines = machines
		mask.Paths = append(mask.Paths, machinesPath)
	}

	// Check and update pools if required.
	if len(pools) > 0 && pools[0] != "" {
		mask.Paths = append(mask.Paths, poolsPath)
		lse.GetChromeosMachineLse().GetDeviceLse().GetDut().Pools = pools
	}

	// Create and assign servo and corresponding masks.
	newServo, paths, err := generateServoWithMask(servo, servoSetup, servoSerial)
	if err != nil {
		return nil, nil, err
	}
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Servo = newServo
	mask.Paths = append(mask.Paths, paths...)

	// Create and assign rpm and corresponding masks.
	rpm, paths := generateRPMWithMask(rpmHost, rpmOutlet)
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Rpm = rpm
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
		if ufsUtil.ContainsAnyStrings(c.tags, utils.ClearFieldValue) {
			lse.Tags = nil
		}
	}

	// Check if nothing is being updated. Updating with an empty mask overwrites everything.
	if !c.forceDeploy && (len(mask.Paths) == 0 || mask.Paths[0] == "") {
		return nil, nil, cmdlib.NewQuietUsageError(c.Flags, "Nothing to update")
	}

	return lse, mask, nil
}

// generateServoWithMask generates a servo object from the given inputs and corresponding mask.
func generateServoWithMask(servo, servoSetup, servoSerial string) (*chromeosLab.Servo, []string, error) {
	// Attempt to parse servo hostname and port.
	servoHost, servoPort, err := parseServoHostnamePort(servo)
	if err != nil {
		return nil, nil, err
	}
	// If servo is being deleted. Return nil with mask path for servo. Ignore other params.
	if servoHost == utils.ClearFieldValue {
		return nil, []string{servoHostPath}, nil
	}

	newServo := &chromeosLab.Servo{}
	// Clear servo_type and servo_topology before deploying. Specifying path only assigns default empty values.
	paths := []string{servoTypePath, servoTopologyPath}
	// Check and update servo port.
	if servoPort != int32(0) {
		paths = append(paths, servoPortPath)
		newServo.ServoPort = servoPort
	}

	if servoSetup != "" {
		paths = append(paths, servoSetupPath)
		sst := chromeosLab.ServoSetupType(chromeosLab.ServoSetupType_value[appendServoSetupPrefix(servoSetup)])
		newServo.ServoSetup = sst
	}

	if servoSerial != "" {
		paths = append(paths, servoSerialPath)
		newServo.ServoSerial = servoSerial
	}

	if servoHost != "" {
		paths = append(paths, servoHostPath)
		newServo.ServoHostname = servoHost
	}
	return newServo, paths, nil
}

// generateRPMWithMask generates a rpm object from the given inputs and corresponding mask.
func generateRPMWithMask(rpmHost, rpmOutlet string) (*chromeosLab.RPM, []string) {
	// Check if rpm is being deleted.
	if rpmHost == utils.ClearFieldValue {
		// Generate mask and empty rpm.
		return nil, []string{rpmHostPath}
	}

	rpm := &chromeosLab.RPM{}
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

// updateDeployActions updates the deploySkipActions based on boolean force options.
func (c *updateDUT) updateDeployActions(actions []string) []string {
	// Append the enforced deploy actions.
	if c.forceDownloadImage && !ufsUtil.ContainsAnyStrings(actions, "stage-usb") {
		actions = append(actions, "stage-usb")
	}
	if c.forceInstallOS && !ufsUtil.ContainsAnyStrings(actions, "install-test-image") {
		actions = append(actions, "install-test-image")
	}
	if c.forceInstallFirmware {
		if !ufsUtil.ContainsAnyStrings(actions, "install-firmware") {
			actions = append(actions, "install-firmware")
		}
		if !ufsUtil.ContainsAnyStrings(actions, "verify-recovery-mode") {
			actions = append(actions, "verify-recovery-mode")
		}
	}
	if (c.forceInstallFirmware || c.forceInstallOS || c.forceUpdateLabels) && !ufsUtil.ContainsAnyStrings(actions, "update-label") {
		actions = append(actions, "update-label")
	}
	return actions
}

// getDeployActions checks the machineLse request and decides actions required for the deploy task.
//
// Actions for deploy task are determined based on the following.
// 1. Updates to servo/rpm will start deploy task with run-pre-deploy-verification.
// 2. Updates to asset will start deploy task with stage-usb, install-test-image, install-firmware,
//    update-label, verify-recovery-mode and run-pre-deploy-verification
// 3. If both are updated then asset takes precedence and actions in (2) are run.
// 4. If neither of them is found. Return nil, nil.
func (c *updateDUT) getDeployActions(ctx context.Context, ic ufsAPI.FleetClient, req *ufsAPI.UpdateMachineLSERequest) (a []string, err error) {
	defer func() {
		// Cannot trust JSON input to have all the fields. Log error.
		if r := recover(); r != nil {
			if c.newSpecsFile != "" && !utils.IsCSVFile(c.newSpecsFile) {
				// JSON update might be missing some fields.
				err = errors.Reason("getDeployActions - Error: %v. Check %s for errors.", r, c.newSpecsFile).Err()
			} else {
				// InternalError. This should not happen.
				err = errors.Reason("getDeployActions - InternalError: %v.", r).Err()
			}
			a = nil
			return
		}
	}()
	// Check if its partial update. Determine actions and state based on what's being updated.
	if req.UpdateMask != nil && len(req.UpdateMask.Paths) > 0 {
		if ufsUtil.ContainsAnyStrings(req.UpdateMask.Paths, "machines") {
			// Asset update. Set state to manual_repair.
			req.MachineLSE.ResourceState = ufspb.State_STATE_DEPLOYED_TESTING
			return assetUpdateDeployActions, nil
		}
		if ufsUtil.ContainsAnyStrings(req.UpdateMask.Paths, partialUpdateDeployPaths...) {
			// RPM/Servo update set state to manual_repair.
			req.MachineLSE.ResourceState = ufspb.State_STATE_DEPLOYED_TESTING
			// Append any options that were set to force and return.
			return partialUpdateDeployActions, nil
		}
		return nil, nil
	}

	// Check if it's a JSON update and validate full update.
	if c.newSpecsFile != "" && !utils.IsCSVFile(c.newSpecsFile) {
		// Full update requires verifying what's being changed on the existing DUT.
		newDut := req.MachineLSE

		// Get the existing DUT configuration.
		oldDut, err := ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
			Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, newDut.GetName()),
		})

		// If DUT doesn't exist return error as update will fail.
		if err != nil {
			return nil, errors.Annotate(err, "getDeployActions - Please check if DUT exists before updating. Failed to get DUT %s", newDut.GetName()).Err()
		}

		// Fail if the target is not a DUT.
		if err := utils.IsDUT(oldDut); err != nil {
			return nil, errors.Annotate(err, "getDeployActions - %s is not a DUT", oldDut.GetName()).Err()
		}

		// Check if asset was updated.
		if oldDut.GetMachines()[0] != newDut.GetMachines()[0] {
			// Asset update. Set state to manual_repair.
			req.MachineLSE.ResourceState = ufspb.State_STATE_DEPLOYED_TESTING
			return assetUpdateDeployActions, nil
		}

		// Check for any servo changes. Need to run a deploy task for the following cases
		// 1. Reset/Delete servo. [newServo == nil || newServo.ServoHostname = ""]
		// 2. Adding a new servo. [oldServo == nil || oldServo.ServoHostname = ""]
		// 3. Clear servo type. [newServo.ServoType == ClearFieldValue]
		// 4. Update servo. [newServo != nil && oldServo != nil]

		var oldServo, newServo *chromeosLab.Servo

		// Check if we are deleting servo.
		newServo = req.MachineLSE.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
		if newServo == nil || newServo.GetServoHostname() == "" {
			// Ensure delete.
			req.MachineLSE.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Servo = nil
			// Servo update set state to manual_repair.
			req.MachineLSE.ResourceState = ufspb.State_STATE_DEPLOYED_TESTING
			return partialUpdateDeployActions, nil
		}

		// Check if the user intends to clear servo type and topology
		if newServo.GetServoType() == utils.ClearFieldValue {
			// Clear servo_type and servo_topology as it will be updated by deploy task
			newServo.ServoType = ""
			newServo.ServoTopology = nil
			// Servo update set state to manual_repair.
			req.MachineLSE.ResourceState = ufspb.State_STATE_DEPLOYED_TESTING
			// Need to run deploy task.
			return partialUpdateDeployActions, nil
		}

		// Check if we are adding a new servo.
		oldServo = oldDut.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
		if oldServo == nil || oldServo.GetServoHostname() == "" {
			// Servo update set state to manual_repair.
			req.MachineLSE.ResourceState = ufspb.State_STATE_DEPLOYED_TESTING
			return partialUpdateDeployActions, nil
		}

		// Check if servo was updated by the user.
		// Make a copy of oldServo for comparison.
		oldServoCopy := proto.Clone(oldServo).(*chromeosLab.Servo)
		// Don't compare servo type or topology as it's not input by the user.
		oldServoCopy.ServoType = ""
		oldServoCopy.ServoTopology = nil
		// Check if the servo host/port/serial is updated.
		if !ufsUtil.ProtoEqual(oldServoCopy, newServo) {
			// Servo update set state to manual_repair.
			req.MachineLSE.ResourceState = ufspb.State_STATE_DEPLOYED_TESTING
			return partialUpdateDeployActions, nil
		}
		// User doesn't intend to update servo. Avoid calling the deploy task and copy servo_type and topology from oldServo.
		newServo.ServoType = oldServo.GetServoType()
		newServo.ServoTopology = oldServo.GetServoTopology()
		req.MachineLSE.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Servo = newServo

		// Check if rpm was updated.
		var oldRpm, newRpm *chromeosLab.RPM
		// Get existing rpm from the DUT.
		if p := oldDut.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals(); p != nil {
			oldRpm = p.GetRpm()
		}
		newRpm = req.MachineLSE.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetRpm()
		// Check if anything in RPM was updated.
		if !ufsUtil.ProtoEqual(oldRpm, newRpm) {
			// RPM update set state to manual_repair.
			req.MachineLSE.ResourceState = ufspb.State_STATE_DEPLOYED_TESTING
			// Append any options that were set to force and return.
			return partialUpdateDeployActions, nil
		}
	}
	// Didn't find any reason to run deploy task.
	return nil, nil
}

// updateDUTToUFS verifies the request and calls UpdateMachineLSE API with the given request.
func (c *updateDUT) updateDUTToUFS(ctx context.Context, ic ufsAPI.FleetClient, req *ufsAPI.UpdateMachineLSERequest) error {
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
	// Remove prefix from the request. It's used for comparison later.
	req.MachineLSE.Name = ufsUtil.RemovePrefix(req.MachineLSE.Name)
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Printf("Successfully updated DUT to UFS: %s \n", res.GetName())
	return nil
}

// deployDUTToSwarming starts a re-deploy task for the given DUT.
func (c *updateDUT) deployDUTToSwarming(ctx context.Context, tc *swarming.TaskCreator, lse *ufspb.MachineLSE, actions []string) error {
	var hostname, machine string
	// Using hostname because name has resource prefix
	hostname = lse.GetHostname()
	machines := lse.GetMachines()
	if len(machines) > 0 {
		machine = machines[0]
	}
	task, err := tc.DeployDut(ctx, hostname, machine, defaultSwarmingPool, c.deployTaskTimeout, actions, c.deployTags, nil)
	if err != nil {
		return err
	}
	fmt.Printf("Triggered Deploy task for DUT %s. Follow the deploy job at %s\n", hostname, task.TaskURL)

	return nil
}
