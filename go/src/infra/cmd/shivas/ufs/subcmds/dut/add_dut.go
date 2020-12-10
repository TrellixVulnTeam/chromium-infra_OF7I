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
	"go.chromium.org/luci/common/errors"
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

const (
	// SwarmingAPIBasePathFormat is the endpoint format for swarming servers.
	SwarmingAPIBasePathFormat = "%s_ah/api/swarming/v1/"
)

// Regexp to enforce the input format
var servoHostPortRegexp = regexp.MustCompile(`^[a-zA-Z0-9\-\.]+:[0-9]+$`)
var defaultDeployTaskActions = []string{"stage-usb", "install-test-image", "update-label", "install-firmware", "verify-recovery-mode", "run-pre-deploy-verification"}

// TODO(anushruth): Find a better place to put these tags.
var shivasTags = []string{"shivas:" + site.VersionNumber, "triggered_using:shivas"}

// AddDUTCmd adds a MachineLSE to the database. And starts a swarming job to deploy.
var AddDUTCmd = &subcommands.Command{
	UsageLine: "dut [options ...]",
	ShortDesc: "Deploy a dut.",
	LongDesc:  cmdhelp.AddDUTLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addDUT{
			pools:            []string{},
			deployTags:       shivasTags,
			deployDimensions: make(map[string]string),
		}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.DUTRegistrationFileText)

		c.Flags.StringVar(&c.hostname, "name", "", "hostname of the DUT.")
		c.Flags.StringVar(&c.machine, "machine", "", "asset tag of the machine.")
		c.Flags.StringVar(&c.servo, "servo", "", "servo hostname and port as hostname:port. (port is assigned by UFS if missing)")
		c.Flags.StringVar(&c.servoSerial, "servo-serial", "", "serial number for the servo")
		c.Flags.Var(flag.StringSlice(&c.pools), "pools", "comma seperated pools assigned to the DUT.")
		c.Flags.StringVar(&c.rpm, "rpm", "", "rpm assigned to the DUT.")
		c.Flags.StringVar(&c.rpmOutlet, "rpm-outlet", "", "rpm outlet used for the DUT.")
		c.Flags.BoolVar(&c.deploy, "deploy", true, "run the deploy task.")
		c.Flags.Int64Var(&c.deployTaskTimeout, "deploy-timeout", swarming.DeployTaskExecutionTimeout, "execution timeout for deploy task in seconds.")
		c.Flags.BoolVar(&c.ufs, "ufs", true, "update UFS with the DUT info.")
		c.deployActions = defaultDeployTaskActions
		c.Flags.Var(flag.StringSlice(&c.deployActions), "deploy-actions", "comma seperated actions for deployment task.")
		c.Flags.Var(flag.StringSlice(&c.deployTags), "deploy-tags", "comma seperated tags for deployment task.")
		c.Flags.Var(flag.StringMap(c.deployDimensions), "deploy-dim", "dimension for deployment (Ex: id:crossk-chromeos1-row4-rack8-host2). 'dut_id:<machine>' and 'pool:<pool>'  dimsensions included by default.")
		return c
	},
}

type addDUT struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string
	hostname     string
	machine      string
	servo        string
	servoSerial  string
	pools        []string
	rpm          string
	rpmOutlet    string

	deploy            bool
	ufs               bool
	deployTaskTimeout int64
	deployActions     []string
	deployDimensions  map[string]string
	deployTags        []string
}

var mcsvFields = []string{
	"name",
	"machine",
	"servo_host",
	"servo_port",
	"servo_serial",
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
		if c.ufs {
			fmt.Printf("Using UFS service %s \n", e.UnifiedFleetService)
		}
		if c.deploy {
			fmt.Printf("Using swarming service %s \n", e.SwarmingService)
		}
	}

	tc, err := swarming.NewTaskCreator(ctx, &c.authFlags, e.SwarmingService)
	if err != nil {
		return err
	}
	tc.LogdogService = e.LogdogService
	tc.SwarmingServiceAccount = e.SwarmingServiceAccount
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	machineLSEs, err := c.parseArgs()
	if err != nil {
		return err
	}
	for _, lse := range machineLSEs {
		if len(lse.GetMachines()) == 0 {
			fmt.Printf("Failed to add DUT %s to UFS. It is not linked to any Asset(Machine).\n", lse.GetName())
			continue
		}
		// Update the UFS database if enabled.
		if c.ufs {
			if err := c.addDutToUFS(ctx, ic, lse); err != nil {
				// skip deployment
				continue
			}
		}
		// Start a swarming deploy task for the DUT.
		if c.deploy {
			c.deployDutToSwarming(ctx, tc, lse)
		}
	}
	if len(machineLSEs) > 1 {
		fmt.Fprintf(a.GetOut(), "\nBatch tasks URL: %s\n\n", tc.SessionTasksURL())
	}
	return nil
}

func (c addDUT) validateArgs() error {
	if !c.deploy && !c.ufs {
		return cmdlib.NewQuietUsageError(c.Flags, "Nothing to do. Choose deploy=true and/or ufs=true")
	}
	if c.ufs && c.newSpecsFile == "" {
		if c.hostname == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Need hostname to create a DUT")
		}
		if c.machine == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Need machine ID to create a DUT")
		}
		if c.servo == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Need servo config to create a DUT")
		}
		if c.rpm == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Need rpm config to create a DUT")
		}
		if c.rpmOutlet == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Need rpm outlet to create a DUT")
		}
		if c.deploy {
			if len(c.pools) == 0 {
				return cmdlib.NewQuietUsageError(c.Flags, "Need at least one pool to deploy the DUT.")
			}
		}
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
	if len(lse.GetChromeosMachineLse().GetDeviceLse().GetDut().Pools) == 0 {
		fmt.Printf("Failed to trigger Deploy task for DUT %s. Need at least one pool to deploy the DUT.\n", lse.GetName())
		return errors.New("no pools in the lse")
	}
	task, err := tc.DeployDut(ctx, lse.GetMachines()[0], lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPools()[0], c.deployTaskTimeout, c.deployActions, c.deployTags, c.deployDimensions)
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
		rpmHost = recMap["rpm_host"]
		rpmOutlet = recMap["rpm_outlet"]
		machines = []string{recMap["machine"]}
		pools = strings.Fields(recMap["pools"])
	} else {
		// command line parameters
		name = c.hostname
		servoHostnamePort := strings.Split(c.servo, ":")
		if len(servoHostnamePort) == 2 {
			servoHost = servoHostnamePort[0]
			port, err := strconv.ParseInt(servoHostnamePort[1], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse servo port %s. %s", servoHostnamePort[1], err)
			}
			servoPort = int32(port)
		} else {
			servoHost = servoHostnamePort[0]
		}
		servoSerial = c.servoSerial
		rpmHost = c.rpm
		rpmOutlet = c.rpmOutlet
		machines = []string{c.machine}
		pools = c.pools
	}
	lse.Name = name
	lse.Hostname = name
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().Hostname = name
	lse.Machines = machines
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().ServoHostname = servoHost
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().ServoPort = servoPort
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().ServoSerial = servoSerial
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetRpm().PowerunitName = rpmHost
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetRpm().PowerunitOutlet = rpmOutlet
	lse.GetChromeosMachineLse().GetDeviceLse().GetDut().Pools = pools
	return lse, nil
}
