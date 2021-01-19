// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"
	"fmt"

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
	lab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

const (
	// Servo related UpdateMask paths.
	servoHostPath   = "dut.servo.hostname"
	servoPortPath   = "dut.servo.port"
	servoSerialPath = "dut.servo.serial"
	servoSetupPath  = "dut.servo.setup"

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
	ShortDesc: "Update a DUT.",
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

	req, err := c.parseArgs()
	if err != nil {
		return err
	}

	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})

	// Collect the deploy actions required for the request. This is done before DUT is changed on UFS.
	actions, err := c.getDeployActions(ctx, ic, req)
	if err != nil {
		return err
	}

	// Attempt to update UFS. Ignore failure if force-deploy.
	if err := c.updateDUTToUFS(ctx, ic, req); err != nil {
		// Return err if deployment is not forced.
		if !c.forceDeploy {
			return err
		}
		fmt.Printf("Failed to update UFS. Attempting to trigger deploy task '-force-deploy'. %s\n", err.Error())
	}

	// Swarm a deploy task if required or enforced.
	if len(actions) > 0 || c.forceDeploy {
		// If deploy task is enforced and we don't need to deploy use partialUpdatedeployActions for deploy.
		if len(actions) == 0 && c.forceDeploy {
			actions = partialUpdateDeployActions
		}

		// Include any enforced actions.
		actions = c.updateDeployActions(actions)

		tc, err := swarming.NewTaskCreator(ctx, &c.authFlags, e.SwarmingService)
		if err != nil {
			return err
		}
		tc.LogdogService = e.LogdogService
		tc.SwarmingServiceAccount = e.SwarmingServiceAccount
		// Start a swarming deploy task for the DUT.
		if err := c.deployDUTToSwarming(ctx, tc, req.GetMachineLSE(), actions); err != nil {
			return err
		}
	}

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
		if _, ok := lab.ServoSetupType_value[appendServoSetupPrefix(c.servoSetupType)]; c.servoSetupType != "" && !ok {
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
func (c *updateDUT) parseArgs() (*ufsAPI.UpdateMachineLSERequest, error) {
	if c.newSpecsFile != "" {
		if utils.IsCSVFile(c.newSpecsFile) {
			return nil, fmt.Errorf("Not implemented yet")
		}
		machineLse := &ufspb.MachineLSE{}
		if err := utils.ParseJSONFile(c.newSpecsFile, machineLse); err != nil {
			return nil, err
		}
		if len(machineLse.GetMachines()) == 0 || machineLse.GetMachines()[0] == "" {
			return nil, cmdlib.NewQuietUsageError(c.Flags, "Missing asset. Use 'machines' to assign asset in json file. Check --help")
		}
		// json input updates without a mask.
		return &ufsAPI.UpdateMachineLSERequest{
			MachineLSE: machineLse,
		}, nil
	}

	lse, mask, err := c.initializeLSEAndMask()
	if err != nil {
		return nil, err
	}
	return &ufsAPI.UpdateMachineLSERequest{
		MachineLSE: lse,
		UpdateMask: mask,
	}, nil
}

// initializeLSEAndMask reads inputs from cmd line inputs and generates LSE and corresponding mask.
func (c *updateDUT) initializeLSEAndMask() (*ufspb.MachineLSE, *field_mask.FieldMask, error) {
	var name, servo, servoSerial, servoSetup, rpmHost, rpmOutlet string
	var machines, pools []string
	// command line parameters
	name = c.hostname
	servo = c.servo
	servoSerial = c.servoSerial
	servoSetup = c.servoSetupType
	rpmHost = c.rpm
	rpmOutlet = c.rpmOutlet
	machines = []string{c.machine}
	pools = c.pools

	// Generate lse and mask
	lse := &ufspb.MachineLSE{
		Lse: &ufspb.MachineLSE_ChromeosMachineLse{
			ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
				ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
					DeviceLse: &ufspb.ChromeOSDeviceLSE{
						Device: &ufspb.ChromeOSDeviceLSE_Dut{
							Dut: &lab.DeviceUnderTest{
								Peripherals: &lab.Peripherals{},
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
func generateServoWithMask(servo, servoSetup, servoSerial string) (*lab.Servo, []string, error) {
	// Attempt to parse servo hostname and port.
	servoHost, servoPort, err := parseServoHostnamePort(servo)
	if err != nil {
		return nil, nil, err
	}
	// If servo is being deleted. Return nil with mask path for servo. Ignore other params.
	if servoHost == utils.ClearFieldValue {
		return nil, []string{servoHostPath}, nil
	}

	newServo := &lab.Servo{}
	paths := []string{}
	// Check and update servo port.
	if servoPort != int32(0) {
		paths = append(paths, servoPortPath)
		newServo.ServoPort = servoPort
	}

	if servoSetup != "" {
		paths = append(paths, servoSetupPath)
		sst := lab.ServoSetupType(lab.ServoSetupType_value[appendServoSetupPrefix(servoSetup)])
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
func generateRPMWithMask(rpmHost, rpmOutlet string) (*lab.RPM, []string) {
	// Check if rpm is being deleted.
	if rpmHost == utils.ClearFieldValue {
		// Generate mask and empty rpm.
		return nil, []string{rpmHostPath}
	}

	rpm := &lab.RPM{}
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
			return nil, errors.Annotate(err, "getDeployActions - Failed to get DUT %s", newDut.GetName()).Err()
		}
		// Check if asset was updated.
		if oldDut.GetMachines()[0] != newDut.GetMachines()[0] {
			// Asset update. Set state to manual_repair.
			req.MachineLSE.ResourceState = ufspb.State_STATE_DEPLOYED_TESTING
			return assetUpdateDeployActions, nil
		}

		// Check if servo was updated.
		var oldServo, newServo *lab.Servo
		// Get existing servo from the DUT and reset ServoType and ServoTopology as they are internally handled
		if p := oldDut.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals(); p != nil {
			oldServo = p.GetServo()
			if oldServo != nil {
				oldServo.ServoType = ""
				oldServo.ServoTopology = nil
			}
		}
		newServo = req.MachineLSE.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
		// Check if anything in servo was updated.
		if !ufsUtil.ProtoEqual(oldServo, newServo) {
			// Servo update set state to manual_repair.
			req.MachineLSE.ResourceState = ufspb.State_STATE_DEPLOYED_TESTING
			// Append any options that were set to force and return.
			return partialUpdateDeployActions, nil
		}
		// Check if rpm was updated.
		var oldRpm, newRpm *lab.RPM
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
