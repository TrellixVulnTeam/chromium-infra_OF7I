// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
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

// Regexp to enforce the input format
var servoHostPortRegexp = regexp.MustCompile(`^[a-zA-Z0-9\-\.]+:[0-9]+$`)
var defaultDeployTaskActions = []string{"servo-verification", "update-label", "verify-recovery-mode", "run-pre-deploy-verification"}

// TODO(anushruth): Find a better place to put these tags.
var shivasTags = []string{"shivas:" + site.VersionNumber, "triggered_using:shivas"}

// defaultPools contains the list of pools used by default.
var defaultPools = []string{"DUT_POOL_QUOTA"}

// defaultSwarmingPool is the swarming pool used for all DUTs.
var defaultSwarmingPool = "ChromeOSSkylab"

// AddDUTCmd adds a MachineLSE to the database. And starts a swarming job to deploy.
var AddDUTCmd = &subcommands.Command{
	UsageLine: "dut [options ...]",
	ShortDesc: "Deploy a DUT",
	LongDesc:  cmdhelp.AddDUTLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addDUT{
			pools:         []string{},
			chameleons:    []string{},
			cameras:       []string{},
			cables:        []string{},
			deployTags:    shivasTags,
			deployActions: defaultDeployTaskActions,
		}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.DUTRegistrationFileText)

		// Asset location fields
		c.Flags.StringVar(&c.zone, "zone", "", "Zone that the asset is in. "+cmdhelp.ZoneFilterHelpText)
		c.Flags.StringVar(&c.rack, "rack", "", "Rack that the asset is in.")

		// DUT/MachineLSE common fields
		c.Flags.StringVar(&c.hostname, "name", "", "hostname of the DUT.")
		c.Flags.StringVar(&c.asset, "asset", "", "asset tag of the machine.")
		c.Flags.StringVar(&c.servo, "servo", "", "servo hostname and port as hostname:port. (port is assigned by UFS if missing)")
		c.Flags.StringVar(&c.servoSerial, "servo-serial", "", "serial number for the servo. Can skip for Servo V3.")
		c.Flags.StringVar(&c.servoSetupType, "servo-setup", "", "servo setup type. Allowed values are "+cmdhelp.ServoSetupTypeAllowedValuesString()+", UFS assigns REGULAR if unassigned.")
		c.Flags.Var(utils.CSVString(&c.pools), "pools", "comma separated pools assigned to the DUT. 'DUT_POOL_QUOTA' is used if nothing is specified")
		c.Flags.Var(utils.CSVString(&c.licenseTypes), "licensetype", cmdhelp.LicenseTypeHelpText)
		c.Flags.Var(utils.CSVString(&c.licenseIds), "licenseid", "the name of the license type. Can specify multiple comma separated values.")
		c.Flags.StringVar(&c.rpm, "rpm", "", "rpm assigned to the DUT.")
		c.Flags.StringVar(&c.rpmOutlet, "rpm-outlet", "", "rpm outlet used for the DUT.")
		c.Flags.Int64Var(&c.deployTaskTimeout, "deploy-timeout", swarming.DeployTaskExecutionTimeout, "execution timeout for deploy task in seconds.")
		c.Flags.BoolVar(&c.ignoreUFS, "ignore-ufs", false, "skip updating UFS create a deploy task.")
		c.Flags.Var(utils.CSVString(&c.deployTags), "deploy-tags", "comma separated tags for deployment task.")
		c.Flags.BoolVar(&c.deploySkipDownloadImage, "deploy-skip-download-image", false, "skips downloading image and staging usb")
		c.Flags.BoolVar(&c.deploySkipInstallFirmware, "deploy-skip-install-fw", false, "skips installing firmware")
		c.Flags.BoolVar(&c.deploySkipInstallOS, "deploy-skip-install-os", false, "skips installing os image")
		c.Flags.StringVar(&c.deploymentTicket, "ticket", "", "the deployment ticket for this machine.")
		c.Flags.Var(utils.CSVString(&c.tags), "tags", "comma separated tags.")
		c.Flags.StringVar(&c.state, "state", "", cmdhelp.StateHelp)
		c.Flags.StringVar(&c.description, "desc", "", "description for the machine.")

		// ACS DUT fields
		c.Flags.Var(utils.CSVString(&c.chameleons), "chameleons", cmdhelp.ChameleonTypeHelpText)
		c.Flags.Var(utils.CSVString(&c.cameras), "cameras", cmdhelp.CameraTypeHelpText)
		c.Flags.Var(utils.CSVString(&c.cables), "cables", cmdhelp.CableTypeHelpText)
		c.Flags.StringVar(&c.antennaConnection, "antennaconnection", "", cmdhelp.AntennaConnectionHelpText)
		c.Flags.StringVar(&c.router, "router", "", cmdhelp.RouterHelpText)
		c.Flags.StringVar(&c.facing, "facing", "", cmdhelp.FacingHelpText)
		c.Flags.StringVar(&c.light, "light", "", cmdhelp.LightHelpText)
		c.Flags.StringVar(&c.carrier, "carrier", "", "name of the carrier.")
		c.Flags.BoolVar(&c.audioBoard, "audioboard", false, "adding this flag will specify if audioboard is present")
		c.Flags.BoolVar(&c.audioBox, "audiobox", false, "adding this flag will specify if audiobox is present")
		c.Flags.BoolVar(&c.atrus, "atrus", false, "adding this flag will specify if atrus is present")
		c.Flags.BoolVar(&c.wifiCell, "wificell", false, "adding this flag will specify if wificell is present")
		c.Flags.BoolVar(&c.touchMimo, "touchmimo", false, "adding this flag will specify if touchmimo is present")
		c.Flags.BoolVar(&c.cameraBox, "camerabox", false, "adding this flag will specify if camerabox is present")
		c.Flags.BoolVar(&c.chaos, "chaos", false, "adding this flag will specify if chaos is present")
		c.Flags.BoolVar(&c.audioCable, "audiocable", false, "adding this flag will specify if audiocable is present")
		c.Flags.BoolVar(&c.smartUSBHub, "smartusbhub", false, "adding this flag will specify if smartusbhub is present")

		// Machine fields
		// crbug.com/1188488 showed us that it might be wise to add model/board during deployment if required.
		c.Flags.StringVar(&c.model, "model", "", "model of the DUT undergoing deployment. If not given, HaRT data is used. Fails if model is not known for the DUT")
		c.Flags.StringVar(&c.board, "board", "", "board of the DUT undergoing deployment. If not given, HaRT data is used. Fails if board is not known for the DUT")
		return c
	},
}

type addDUT struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile   string
	hostname       string
	asset          string
	servo          string
	servoSerial    string
	servoSetupType string
	licenseTypes   []string
	licenseIds     []string
	pools          []string
	rpm            string
	rpmOutlet      string

	ignoreUFS                 bool
	deployTaskTimeout         int64
	deployActions             []string
	deployTags                []string
	deploySkipDownloadImage   bool
	deploySkipInstallOS       bool
	deploySkipInstallFirmware bool
	deploymentTicket          string
	tags                      []string
	state                     string
	description               string

	// Asset location fields
	zone string
	rack string

	// ACS DUT fields
	chameleons        []string
	cameras           []string
	antennaConnection string
	router            string
	cables            []string
	facing            string
	light             string
	carrier           string
	audioBoard        bool
	audioBox          bool
	atrus             bool
	wifiCell          bool
	touchMimo         bool
	cameraBox         bool
	chaos             bool
	audioCable        bool
	smartUSBHub       bool

	// Machine specific fields
	model string
	board string
}

var mcsvFields = []string{
	"name",
	"asset",
	"model",
	"board",
	"servo_host",
	"servo_port",
	"servo_serial",
	"servo_setup",
	"rpm_host",
	"rpm_outlet",
	"pools",
}

// dutDeployUFSParams contains all the data that are needed for deployment of a single DUT
// Asset and its update paths are required here to update location, model and board for the DUT
// See: crbug.com/1188488 for why model and board need to be updated.
type dutDeployUFSParams struct {
	DUT   *ufspb.MachineLSE // MachineLSE of the DUT to be updated
	Asset *ufspb.Asset      // Asset underlying the DUT being updated
	Paths []string          // Update paths for the Asset being updated
}

func (c *addDUT) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addDUT) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
		if !c.ignoreUFS {
			fmt.Printf("Using UFS service %s \n", e.UnifiedFleetService)
		}
		fmt.Printf("Using swarming service %s \n", e.SwarmingService)
	}

	tc, err := swarming.NewTaskCreator(ctx, &c.authFlags, e.SwarmingService)
	if err != nil {
		return err
	}
	tc.LogdogService = e.LogdogService
	tc.SwarmingServiceAccount = e.SwarmingServiceAccount

	dutParams, err := c.parseArgs()
	if err != nil {
		return err
	}

	c.updateDeployActions()

	// Update the UFS database if enabled.
	if !c.ignoreUFS {
		ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
			C:       hc,
			Host:    e.UnifiedFleetService,
			Options: site.DefaultPRPCOptions,
		})

		for _, param := range dutParams {
			if len(param.DUT.GetMachines()) == 0 {
				fmt.Printf("Failed to add DUT %s to UFS. It is not linked to any Asset(Machine).\n", param.DUT.GetName())
				continue
			}
			if err := c.addDutToUFS(ctx, ic, param); err != nil {
				fmt.Printf("Failed to add DUT %s to UFS. Skipping deployment. %s", param.DUT.GetName(), err.Error())
				// skip deployment
				continue
			}
			c.deployDutToSwarming(ctx, tc, param.DUT)
		}
		if len(dutParams) > 1 {
			fmt.Fprintf(a.GetOut(), "\nBatch tasks URL: %s\n\n", tc.SessionTasksURL())
		}
		return nil
	}

	// Run the deployment task
	for _, param := range dutParams {
		c.deployDutToSwarming(ctx, tc, param.DUT)
	}
	return nil
}

func (c addDUT) validateArgs() error {
	if !c.ignoreUFS && c.newSpecsFile == "" {
		if c.asset == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Need asset ID to create a DUT")
		}
		if c.servo == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Need servo config to create a DUT")
		}
		if c.servo != "" {
			// If the servo is not servo V3. Then servo serial is needed
			host, _, err := parseServoHostnamePort(c.servo)
			if err != nil {
				return err
			}
			if !ufsUtil.ServoV3HostnameRegex.MatchString(host) && c.servoSerial == "" {
				return cmdlib.NewQuietUsageError(c.Flags, "Cannot skip servo serial. Not a servo V3 device.")
			}
		}
		if c.servoSetupType != "" {
			if _, ok := chromeosLab.ServoSetupType_value[appendServoSetupPrefix(c.servoSetupType)]; !ok {
				return cmdlib.NewQuietUsageError(c.Flags, "Invalid servo setup %s", c.servoSetupType)
			}
		}
		if (c.rpm != "" && c.rpmOutlet == "") || (c.rpm == "" && c.rpmOutlet != "") {
			return cmdlib.NewQuietUsageError(c.Flags, "Need both rpm and its outlet. %s:%s is invalid", c.rpm, c.rpmOutlet)
		}
		if c.zone != "" && !ufsUtil.IsUFSZone(ufsUtil.RemoveZonePrefix(c.zone)) {
			return cmdlib.NewQuietUsageError(c.Flags, "Invalid zone %s", c.zone)
		}
		if len(c.licenseTypes) != len(c.licenseIds) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNumber of -licensetype(%s) and -licenseid(%s) must be same.", c.licenseTypes, c.licenseIds)
		}
		for _, cp := range c.licenseTypes {
			if !ufsUtil.IsLicenseType(cp) {
				return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid license type name, please check help info for '-licensetype'.", cp)
			}
		}
		for _, cp := range c.chameleons {
			if !ufsUtil.IsChameleonType(cp) {
				return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid chameleon type name, please check help info for '-chameleons'.", cp)
			}
		}
		for _, cp := range c.cameras {
			if !ufsUtil.IsCameraType(cp) {
				return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid camera type name, please check help info for '-cameras'.", cp)
			}
		}
		for _, cp := range c.cables {
			if !ufsUtil.IsCableType(cp) {
				return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid cable type name, please check help info for '-cables'.", cp)
			}
		}
		if c.antennaConnection != "" && !ufsUtil.IsAntennaConnection(c.antennaConnection) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid antenna connection name, please check help info for '-antennaconnection'.", c.antennaConnection)
		}
		if c.router != "" && !ufsUtil.IsRouter(c.router) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid router name, please check help info for '-router'.", c.router)
		}
		if c.facing != "" && !ufsUtil.IsFacing(c.facing) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid facing name, please check help info for '-facing'.", c.facing)
		}
		if c.light != "" && !ufsUtil.IsLight(c.light) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid light name, please check help info for '-light'.", c.light)
		}
	}
	if c.newSpecsFile == "" && c.hostname == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Need hostname to create a DUT")
	}
	return nil
}

func (c *addDUT) parseArgs() ([]*dutDeployUFSParams, error) {
	if c.newSpecsFile != "" {
		if utils.IsCSVFile(c.newSpecsFile) {
			return c.parseMCSV()
		}
		machinelse := &ufspb.MachineLSE{}
		if err := utils.ParseJSONFile(c.newSpecsFile, machinelse); err != nil {
			return nil, err
		}
		asset, paths, err := c.initializeAssetUpdate(machinelse.GetName(), machinelse.GetMachines()[0], "", "")
		if err != nil {
			return nil, err
		}
		return []*dutDeployUFSParams{{
			DUT:   machinelse,
			Asset: asset,
			Paths: paths,
		}}, nil
	}
	// command line parameters
	dutParams, err := c.initializeLSEAndAsset(nil)
	if err != nil {
		return nil, err
	}
	return []*dutDeployUFSParams{dutParams}, nil
}

// parseMCSV parses the MCSV file and returns MachineLSEs
func (c *addDUT) parseMCSV() ([]*dutDeployUFSParams, error) {
	records, err := utils.ParseMCSVFile(c.newSpecsFile)
	if err != nil {
		return nil, err
	}
	var dutParams []*dutDeployUFSParams
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
		p, err := c.initializeLSEAndAsset(recMap)
		if err != nil {
			fmt.Printf("Error [%s:%v]: %v. Skipping add on this line\n", c.newSpecsFile, i+1, err.Error())
		} else {
			dutParams = append(dutParams, p)
		}
	}
	return dutParams, nil
}

func (c *addDUT) addDutToUFS(ctx context.Context, ic ufsAPI.FleetClient, param *dutDeployUFSParams) error {
	// Attempt to update the changes to asset first.
	if err := c.updateAssetToUFS(ctx, ic, param.Asset, param.Paths); err != nil {
		return err
	}
	if !ufsUtil.ValidateTags(param.DUT.Tags) {
		return fmt.Errorf(ufsAPI.InvalidTags)
	}
	res, err := ic.CreateMachineLSE(ctx, &ufsAPI.CreateMachineLSERequest{
		MachineLSE:   param.DUT,
		MachineLSEId: param.DUT.GetName(),
	})
	if err != nil {
		fmt.Printf("Failed to add DUT %s to UFS. UFS add failed %s\n", param.DUT.GetName(), err)
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Printf("Successfully added DUT to UFS: %s \n", res.GetName())
	return nil
}

func (c *addDUT) deployDutToSwarming(ctx context.Context, tc *swarming.TaskCreator, lse *ufspb.MachineLSE) error {
	task, err := tc.DeployDut(ctx, lse.Name, lse.GetMachines()[0], defaultSwarmingPool, c.deployTaskTimeout, c.deployActions, c.deployTags, nil)
	if err != nil {
		fmt.Printf("Failed to trigger Deploy task for DUT %s. Deploy failed %s\n", lse.GetName(), err)
		return err
	}
	fmt.Printf("Triggered Deploy task for DUT %s. Follow the deploy job at %s\n", lse.GetName(), task.TaskURL)
	return nil
}

func (c *addDUT) initializeLSEAndAsset(recMap map[string]string) (*dutDeployUFSParams, error) {
	lse := &ufspb.MachineLSE{
		Lse: &ufspb.MachineLSE_ChromeosMachineLse{
			ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
				ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
					DeviceLse: &ufspb.ChromeOSDeviceLSE{
						Device: &ufspb.ChromeOSDeviceLSE_Dut{
							Dut: &chromeosLab.DeviceUnderTest{
								Peripherals: &chromeosLab.Peripherals{
									Chameleon:     &chromeosLab.Chameleon{},
									Servo:         &chromeosLab.Servo{},
									Rpm:           &chromeosLab.OSRPM{},
									Audio:         &chromeosLab.Audio{},
									Wifi:          &chromeosLab.Wifi{},
									Touch:         &chromeosLab.Touch{},
									CameraboxInfo: &chromeosLab.Camerabox{},
								},
							},
						},
					},
				},
			},
		},
	}
	var name, servoHost, servoSerial, rpmHost, rpmOutlet, model, board string
	var pools, machines []string
	var servoPort int32
	var servoSetup chromeosLab.ServoSetupType
	resourceState := ufsUtil.ToUFSState(c.state)
	if recMap != nil {
		// CSV map
		name = recMap["name"]
		servoHost = recMap["servo_host"]
		if recMap["servo_port"] != "" {
			port, err := strconv.ParseInt(recMap["servo_port"], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse servo port %s. %s", recMap["servo_port"], err)
			}
			servoPort = int32(port)
		}
		servoSerial = recMap["servo_serial"]
		// Check if the host is servo V3. Need servo serial otherwise.
		if !ufsUtil.ServoV3HostnameRegex.MatchString(servoHost) && servoSerial == "" {
			return nil, fmt.Errorf("Not a servo V3 host[%s]. Need servo serial", servoHost)
		}
		sst, ok := chromeosLab.ServoSetupType_value[appendServoSetupPrefix(recMap["servo_setup"])]
		if !ok && recMap["servo_setup"] != "" {
			return nil, fmt.Errorf("Invalid servo setup %s. Valid types are %s", recMap["servo_setup"], cmdhelp.ServoSetupTypeAllowedValuesString())
		}
		servoSetup = chromeosLab.ServoSetupType(sst) // Default value is REGULAR(0).
		rpmHost = recMap["rpm_host"]
		rpmOutlet = recMap["rpm_outlet"]
		machines = []string{recMap["asset"]}
		pools = strings.Fields(recMap["pools"])
		model = recMap["model"]
		board = recMap["board"]
	} else {
		// command line parameters
		name = c.hostname
		var err error
		servoHost, servoPort, err = parseServoHostnamePort(c.servo)
		if err != nil {
			return nil, err
		}
		servoSerial = c.servoSerial
		servoSetup = chromeosLab.ServoSetupType(chromeosLab.ServoSetupType_value[appendServoSetupPrefix(c.servoSetupType)])
		rpmHost = c.rpm
		rpmOutlet = c.rpmOutlet
		machines = []string{c.asset}
		pools = c.pools
		model = c.model
		board = c.board
	}
	lse.Name = name
	lse.Hostname = name
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().Hostname = name
	lse.Machines = machines

	// Use the input params if available for all the options.
	lse.Description = c.description
	lse.DeploymentTicket = c.deploymentTicket
	lse.Tags = c.tags
	lse.ResourceState = resourceState

	licenses := make([]*chromeosLab.License, 0, len(c.licenseTypes))
	for i := range c.licenseTypes {
		licenses = append(licenses, &chromeosLab.License{
			Type:       ufsUtil.ToLicenseType(c.licenseTypes[i]),
			Identifier: c.licenseIds[i],
		})
	}
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().Licenses = licenses
	peripherals := lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
	peripherals.GetServo().ServoHostname = servoHost
	peripherals.GetServo().ServoPort = servoPort
	peripherals.GetServo().ServoSerial = servoSerial
	peripherals.GetServo().ServoSetup = servoSetup
	peripherals.GetRpm().PowerunitName = rpmHost
	peripherals.GetRpm().PowerunitOutlet = rpmOutlet
	if len(pools) > 0 && pools[0] != "" {
		lse.GetChromeosMachineLse().GetDeviceLse().GetDut().Pools = pools
	} else {
		lse.GetChromeosMachineLse().GetDeviceLse().GetDut().Pools = defaultPools
	}

	// ACS DUT fields
	chameleons := make([]chromeosLab.ChameleonType, 0, len(c.chameleons))
	for _, cp := range c.chameleons {
		chameleons = append(chameleons, ufsUtil.ToChameleonType(cp))
	}
	cameras := make([]*chromeosLab.Camera, 0, len(c.cameras))
	for _, cp := range c.cameras {
		camera := &chromeosLab.Camera{
			CameraType: ufsUtil.ToCameraType(cp),
		}
		cameras = append(cameras, camera)
	}
	cables := make([]*chromeosLab.Cable, 0, len(c.cables))
	for _, cp := range c.cables {
		cable := &chromeosLab.Cable{
			Type: ufsUtil.ToCableType(cp),
		}
		cables = append(cables, cable)
	}
	peripherals.GetChameleon().ChameleonPeripherals = chameleons
	peripherals.ConnectedCamera = cameras
	peripherals.Cable = cables
	peripherals.GetWifi().AntennaConn = ufsUtil.ToAntennaConnection(c.antennaConnection)
	peripherals.GetWifi().Router = ufsUtil.ToRouter(c.router)
	peripherals.GetCameraboxInfo().Facing = ufsUtil.ToFacing(c.facing)
	peripherals.GetCameraboxInfo().Light = ufsUtil.ToLight(c.light)
	peripherals.GetChameleon().AudioBoard = c.audioBoard
	peripherals.GetAudio().AudioBox = c.audioBox
	peripherals.GetAudio().Atrus = c.atrus
	peripherals.GetAudio().AudioCable = c.audioCable
	peripherals.GetWifi().Wificell = c.wifiCell
	peripherals.GetTouch().Mimo = c.touchMimo
	peripherals.Carrier = c.carrier
	peripherals.Camerabox = c.cameraBox
	peripherals.Chaos = c.chaos
	peripherals.SmartUsbhub = c.smartUSBHub
	// Get the updated asset and update paths
	asset, paths, err := c.initializeAssetUpdate(lse.GetName(), machines[0], model, board)
	if err != nil {
		return nil, err
	}
	return &dutDeployUFSParams{
		DUT:   lse,
		Asset: asset,
		Paths: paths,
	}, nil
}

// initializeAssetUpdate initializes asset proto and update mask for updating the asset
func (c *addDUT) initializeAssetUpdate(hostname, machine, model, board string) (*ufspb.Asset, []string, error) {
	loc, err := utils.GetLocation(hostname)
	if err != nil {
		fmt.Printf("Failed to update asset location for DUT %s in UFS.\n", hostname)
		return nil, nil, err
	}
	asset := &ufspb.Asset{
		Name:     ufsUtil.AddPrefix(ufsUtil.AssetCollection, machine),
		Location: loc,
		Info:     &ufspb.AssetInfo{},
	}
	// Create the update field mask.
	var paths []string
	// Override zone info with user provided option.
	if c.zone != "" {
		asset.GetLocation().Zone = ufsUtil.ToUFSZone(c.zone)
	}
	// Override rack info with user provided option.
	if c.rack != "" {
		asset.GetLocation().Rack = c.rack
	}
	// Override model with user provided option.
	if model != "" {
		asset.Model = model
		asset.GetInfo().Model = model
		paths = append(paths, "model")
	}
	// Override board with user provided option
	if board != "" {
		asset.GetInfo().BuildTarget = board
		paths = append(paths, "info.build_target")
	}
	if asset.GetLocation().GetZone() != ufspb.Zone_ZONE_UNSPECIFIED {
		paths = append(paths, "location.zone")
		asset.Realm = ufsUtil.ToUFSRealm(asset.GetLocation().GetZone().String())
	}
	if asset.GetLocation().GetRack() != "" {
		paths = append(paths, "location.rack")
	}
	if asset.GetLocation().GetRackNumber() != "" {
		paths = append(paths, "location.rack_number")
	}
	if asset.GetLocation().GetRow() != "" {
		paths = append(paths, "location.row")
	}
	if asset.GetLocation().GetPosition() != "" {
		paths = append(paths, "location.position")
	}
	if asset.GetLocation().GetBarcodeName() != "" {
		paths = append(paths, "location.barcode_name")
	}
	return asset, paths, nil
}

func parseServoHostnamePort(servo string) (string, int32, error) {
	var servoHostname string
	var servoPort int32
	servoHostnamePort := strings.Split(servo, ":")
	if len(servoHostnamePort) == 2 {
		servoHostname = servoHostnamePort[0]
		port, err := strconv.ParseInt(servoHostnamePort[1], 10, 32)
		if err != nil {
			return "", int32(0), err
		}
		servoPort = int32(port)
	} else {
		servoHostname = servoHostnamePort[0]
	}
	return servoHostname, servoPort, nil
}

// updateDeployActions updates the deploySkipActions based on boolean skip options
func (c *addDUT) updateDeployActions() {
	if !c.deploySkipDownloadImage {
		c.deployActions = append(c.deployActions, "stage-usb")
	}
	if !c.deploySkipInstallOS {
		c.deployActions = append(c.deployActions, "install-test-image")
	}
	if !c.deploySkipInstallFirmware {
		c.deployActions = append(c.deployActions, "install-firmware")
	}
}

func appendServoSetupPrefix(servoSetup string) string {
	return fmt.Sprintf("SERVO_SETUP_%s", servoSetup)
}

// updateAssetToUFS calls UpdateAsset API in UFS with asset and partial paths
func (c *addDUT) updateAssetToUFS(ctx context.Context, ic ufsAPI.FleetClient, asset *ufspb.Asset, paths []string) error {
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
