package dut

import (
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
	swarming "infra/libs/cros/swarming"
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
		c.Flags.StringVar(&c.hostname, "name", "", "hostname of the DUT.")
		c.Flags.StringVar(&c.machine, "machine", "", "asset tag of the machine.")
		c.Flags.StringVar(&c.servo, "servo", "", "servo hostname and port as hostname:port. (port is assigned by UFS if missing)")
		c.Flags.StringVar(&c.servoSerial, "servo-serial", "", "serial number for the servo")
		c.Flags.Var(flag.StringSlice(&c.pools), "pools", "comma seperated pools assigned to the DUT.")
		c.Flags.StringVar(&c.rpm, "rpm", "", "rpm assigned to the DUT.")
		c.Flags.StringVar(&c.rpmOutlet, "rpm-outlet", "", "rpm outlet used for the DUT.")
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.DUTRegistrationFileText)
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

	// Update the UFS database if enabled.
	if c.ufs {
		ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
			C:       hc,
			Host:    e.UnifiedFleetService,
			Options: site.DefaultPRPCOptions,
		})

		machineLSE, err := c.parseArgs()
		if err != nil {
			return err
		}

		var machineLSERequest ufsAPI.CreateMachineLSERequest

		machineLSERequest.MachineLSEId = c.hostname
		machineLSERequest.MachineLSE = machineLSE

		res, err := ic.CreateMachineLSE(ctx, &machineLSERequest)
		if err != nil {
			return err
		}
		res.Name = ufsUtil.RemovePrefix(res.Name)
		utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
		fmt.Printf("Successfully added DUT to UFS: %s \n", res.GetName())
	}

	// Start a swarming deploy task for the DUT.
	if c.deploy {
		task, err := tc.DeployDut(ctx, c.machine, c.pools[0], c.deployTaskTimeout, c.deployActions, c.deployTags, c.deployDimensions)
		if err != nil {
			return err
		}
		fmt.Printf("Triggered Deploy task for DUT %s. Follow the deploy job at %s\n", c.hostname, task.TaskURL)
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
	}
	if c.deploy {
		if len(c.pools) == 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Need at least one pool to deploy the DUT.")
		}
	}
	return nil
}

func (c *addDUT) parseArgs() (*ufspb.MachineLSE, error) {
	machineLSE := &ufspb.MachineLSE{}
	if c.newSpecsFile != "" {
		if err := utils.ParseJSONFile(c.newSpecsFile, machineLSE); err != nil {
			return nil, err
		}
		return machineLSE, nil
	}

	machineLSE.Hostname = c.hostname
	machineLSE.Machines = []string{c.machine}

	var servoHostname string
	var servoPort int32
	servoHostnamePort := strings.Split(c.servo, ":")
	if len(servoHostnamePort) == 2 {
		servoHostname = servoHostnamePort[0]
		port, err := strconv.ParseInt(servoHostnamePort[1], 10, 32)
		if err != nil {
			return nil, err
		}
		servoPort = int32(port)
	} else {
		servoHostname = servoHostnamePort[0]
	}

	machineLSE.Lse = &ufspb.MachineLSE_ChromeosMachineLse{
		ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
			ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
				DeviceLse: &ufspb.ChromeOSDeviceLSE{
					Device: &ufspb.ChromeOSDeviceLSE_Dut{
						Dut: &lab.DeviceUnderTest{
							Hostname: c.hostname,
							Peripherals: &lab.Peripherals{
								Servo: &lab.Servo{
									ServoHostname: servoHostname,
									ServoPort:     servoPort,
									ServoSerial:   c.servoSerial,
								},
								Rpm: &lab.RPM{
									PowerunitName:   c.rpm,
									PowerunitOutlet: c.rpmOutlet,
								},
							},
							Pools: c.pools,
						},
					},
				},
			},
		},
	}
	return machineLSE, nil
}
