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
	"go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/grpc/prpc"

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

// Regexp to enforce the input format
var servoHostPortRegexp = regexp.MustCompile(`^[a-zA-Z0-9\-\.]+:[0-9]+$`)
var defaultDeployTaskActions = []string{"update-label", "verify-recovery-mode", "run-pre-deploy-verification"}

// TODO(anushruth): Find a better place to put these tags.
var shivasTags = []string{"shivas:" + site.VersionNumber, "triggered_using:shivas"}

// defaultPools contains the list of critical pools used by default.
var defaultPools = []string{"DUT_POOL_QUOTA"}

// AddDUTCmd adds a MachineLSE to the database. And starts a swarming job to deploy.
var AddDUTCmd = &subcommands.Command{
	UsageLine: "dut [options ...]",
	ShortDesc: "Deploy a dut.",
	LongDesc:  cmdhelp.AddDUTLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addDUT{
			pools:         defaultPools,
			deployTags:    shivasTags,
			deployActions: defaultDeployTaskActions,
		}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.DUTRegistrationFileText)

		c.Flags.StringVar(&c.hostname, "name", "", "hostname of the DUT.")
		c.Flags.StringVar(&c.machine, "machine", "", "asset tag of the machine.")
		c.Flags.StringVar(&c.servo, "servo", "", "servo hostname and port as hostname:port. (port is assigned by UFS if missing)")
		c.Flags.StringVar(&c.servoSerial, "servo-serial", "", "serial number for the servo. Can skip for Servo V3.")
		c.Flags.StringVar(&c.servoSetupType, "servo-setup", "", "servo setup type. Allowed values are "+cmdhelp.ServoSetupTypeAllowedValuesString()+", UFS assigns SERVO_SETUP_REGULAR if unassigned.")
		c.Flags.Var(flag.StringSlice(&c.pools), "pools", "comma seperated pools assigned to the DUT. Enclose in double quotes (\") for list of pools. Allowed values are "+cmdhelp.CriticalPoolsAllowedValuesString()+".")
		c.Flags.StringVar(&c.rpm, "rpm", "", "rpm assigned to the DUT.")
		c.Flags.StringVar(&c.rpmOutlet, "rpm-outlet", "", "rpm outlet used for the DUT.")
		c.Flags.Int64Var(&c.deployTaskTimeout, "deploy-timeout", swarming.DeployTaskExecutionTimeout, "execution timeout for deploy task in seconds.")
		c.Flags.BoolVar(&c.ignoreUFS, "ignore-ufs", false, "skip updating UFS create a deploy task.")
		c.Flags.Var(flag.StringSlice(&c.deployTags), "deploy-tags", "comma seperated tags for deployment task. Enclose in double quotes (\") for list of tags")
		c.Flags.BoolVar(&c.deploySkipDownloadImage, "deploy-skip-download-image", false, "skips downloading image and staging usb")
		c.Flags.BoolVar(&c.deploySkipInstallFirmware, "deploy-skip-install-fw", false, "skips installing firmware")
		c.Flags.BoolVar(&c.deploySkipInstallOS, "deploy-skip-install-os", false, "skips installing os image")
		c.Flags.StringVar(&c.deploymentTicket, "ticket", "", "the deployment ticket for this machine.")
		c.Flags.Var(flag.StringSlice(&c.tags), "tags", "comma separated tags. Enclose in double quotes (\") for list of tags.")
		c.Flags.StringVar(&c.state, "state", "", cmdhelp.StateHelp)
		c.Flags.StringVar(&c.description, "desc", "", "description for the machine.")
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
	machine        string
	servo          string
	servoSerial    string
	servoSetupType string
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
}

var mcsvFields = []string{
	"name",
	"machine",
	"servo_host",
	"servo_port",
	"servo_serial",
	"servo_setup",
	"rpm_host",
	"rpm_outlet",
	"pools",
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

	machineLSEs, err := c.parseArgs()
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

		for _, lse := range machineLSEs {
			if len(lse.GetMachines()) == 0 {
				fmt.Printf("Failed to add DUT %s to UFS. It is not linked to any Asset(Machine).\n", lse.GetName())
				continue
			}
			if err := c.addDutToUFS(ctx, ic, lse); err != nil {
				// skip deployment
				continue
			}
			c.deployDutToSwarming(ctx, tc, lse)
		}
		if len(machineLSEs) > 1 {
			fmt.Fprintf(a.GetOut(), "\nBatch tasks URL: %s\n\n", tc.SessionTasksURL())
		}
		return nil
	}

	// Run the deployment task
	for _, lse := range machineLSEs {
		c.deployDutToSwarming(ctx, tc, lse)
	}
	return nil
}

func (c addDUT) validateArgs() error {
	if !c.ignoreUFS && c.newSpecsFile == "" {
		if c.machine == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Need machine ID to create a DUT")
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
			if _, ok := lab.ServoSetupType_value[c.servoSetupType]; !ok {
				return cmdlib.NewQuietUsageError(c.Flags, "Invalid servo setup %s", c.servoSetupType)
			}
		}
		if (c.rpm != "" && c.rpmOutlet == "") || (c.rpm == "" && c.rpmOutlet != "") {
			return cmdlib.NewQuietUsageError(c.Flags, "Need both rpm and its outlet. %s:%s is invalid", c.rpm, c.rpmOutlet)
		}
		if len(c.pools) != 0 {
			for _, name := range c.pools {
				if _, ok := lab.DeviceUnderTest_DUTPool_value[name]; !ok {
					return cmdlib.NewQuietUsageError(c.Flags, "Invalid pool %s, Valid pools are %s.", name, cmdhelp.CriticalPoolsAllowedValuesString())
				}
			}
		}
	}
	if c.hostname == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Need hostname to create a DUT")
	}
	return nil
}

func (c *addDUT) parseArgs() ([]*ufspb.MachineLSE, error) {
	if c.newSpecsFile != "" {
		if utils.IsCSVFile(c.newSpecsFile) {
			return c.parseMCSV()
		}
		machinelse := &ufspb.MachineLSE{}
		if err := utils.ParseJSONFile(c.newSpecsFile, machinelse); err != nil {
			return nil, err
		}
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
func (c *addDUT) parseMCSV() ([]*ufspb.MachineLSE, error) {
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

func (c *addDUT) addDutToUFS(ctx context.Context, ic ufsAPI.FleetClient, lse *ufspb.MachineLSE) error {
	res, err := ic.CreateMachineLSE(ctx, &ufsAPI.CreateMachineLSERequest{
		MachineLSE:   lse,
		MachineLSEId: lse.GetName(),
	})
	if err != nil {
		fmt.Printf("Failed to add DUT %s to UFS. UFS add failed %s\n", lse.GetName(), err)
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Printf("Successfully added DUT to UFS: %s \n", res.GetName())
	return nil
}

func (c *addDUT) deployDutToSwarming(ctx context.Context, tc *swarming.TaskCreator, lse *ufspb.MachineLSE) error {
	task, err := tc.DeployDut(ctx, lse.Name, lse.GetMachines()[0], lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPools()[0], c.deployTaskTimeout, c.deployActions, c.deployTags, nil)
	if err != nil {
		fmt.Printf("Failed to trigger Deploy task for DUT %s. Deploy failed %s\n", lse.GetName(), err)
		return err
	}
	fmt.Printf("Triggered Deploy task for DUT %s. Follow the deploy job at %s\n", lse.GetName(), task.TaskURL)
	return nil
}

func (c *addDUT) initializeLSE(recMap map[string]string) (*ufspb.MachineLSE, error) {
	lse := &ufspb.MachineLSE{
		Lse: &ufspb.MachineLSE_ChromeosMachineLse{
			ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
				ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
					DeviceLse: &ufspb.ChromeOSDeviceLSE{
						Device: &ufspb.ChromeOSDeviceLSE_Dut{
							Dut: &lab.DeviceUnderTest{
								Peripherals: &lab.Peripherals{
									Servo: &lab.Servo{},
									Rpm:   &lab.RPM{},
								},
							},
						},
					},
				},
			},
		},
	}
	var name, servoHost, servoSerial, rpmHost, rpmOutlet string
	var pools, machines []string
	var servoPort int32
	var servoSetup lab.ServoSetupType
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
		if _, ok := lab.ServoSetupType_value[recMap["servo_setup"]]; !ok {
			return nil, fmt.Errorf("Invalid servo setup %s. Valid types are %s", recMap["servo_setup"], cmdhelp.ServoSetupTypeAllowedValuesString())
		}
		servoSetup = lab.ServoSetupType(lab.ServoSetupType_value[recMap["servo_setup"]])
		rpmHost = recMap["rpm_host"]
		rpmOutlet = recMap["rpm_outlet"]
		machines = []string{recMap["machine"]}
		pools = strings.Fields(recMap["pools"])
	} else {
		// command line parameters
		name = c.hostname
		var err error
		servoHost, servoPort, err = parseServoHostnamePort(c.servo)
		if err != nil {
			return nil, err
		}
		servoSerial = c.servoSerial
		servoSetup = lab.ServoSetupType(lab.ServoSetupType_value[c.servoSetupType])
		rpmHost = c.rpm
		rpmOutlet = c.rpmOutlet
		machines = []string{c.machine}
		pools = c.pools
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

	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().ServoHostname = servoHost
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().ServoPort = servoPort
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().ServoSerial = servoSerial
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().ServoSetup = servoSetup
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetRpm().PowerunitName = rpmHost
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetRpm().PowerunitOutlet = rpmOutlet
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().Pools = []string{"ChromeOSSkylab"}
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().CriticalPools = parsePools(pools)
	return lse, nil
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

func parsePools(criticalPools []string) []lab.DeviceUnderTest_DUTPool {
	if len(criticalPools) == 0 {
		return nil
	}
	pools := []lab.DeviceUnderTest_DUTPool{}
	for _, name := range criticalPools {
		pools = append(pools, lab.DeviceUnderTest_DUTPool(lab.DeviceUnderTest_DUTPool_value[name]))
	}
	return pools
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
