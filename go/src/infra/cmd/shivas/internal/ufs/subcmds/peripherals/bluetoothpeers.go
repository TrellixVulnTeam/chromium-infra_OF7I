// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package peripherals

import (
	"fmt"
	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	lab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	rpc "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/util"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/grpc/prpc"
)

var (
	AddBluetoothPeersCmd     = btpsCmd(actionAdd)
	ReplaceBluetoothPeersCmd = btpsCmd(actionReplace)
	DeleteBluetoothPeersCmd  = btpsCmd(actionDelete)
)

type action int

const (
	actionAdd action = iota
	actionReplace
	actionDelete
)

// btpsCmd creates command for adding, removing, or completely replacing Bluetooth peers on a DUT.
func btpsCmd(mode action) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "bluetooth-peers -dut {DUT name} -hostname {BTP hostname} [-hostname {BTP hostname}...]",
		ShortDesc: "Manage Bluetooth peers connect to a DUT",
		LongDesc:  cmdhelp.ManageBTPsLongDesc,
		CommandRun: func() subcommands.CommandRun {
			c := manageBTPsCmd{mode: mode}
			c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
			c.envFlags.Register(&c.Flags)
			c.commonFlags.Register(&c.Flags)

			c.Flags.StringVar(&c.dutName, "dut", "", "DUT name to update")
			c.Flags.Var(flag.StringSlice(&c.hostnames), "hostname", "hostname for Bluetooth peer, can be specified multiple times")

			return &c
		},
	}
}

// manageBTPsCmd supports adding, replacing, or deleting BTPs.
type manageBTPsCmd struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	dutName   string
	hostnames []string
	hostsMap  map[string]bool // set of hostnames

	mode action
}

// Run executed the BTP management subcommand. It cleans up passed flags and validates them.
func (c *manageBTPsCmd) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.run(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

// run implements the core logic for Run.
func (c *manageBTPsCmd) run(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.cleanAndValidateFlags(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	ctx = utils.SetupContext(ctx, util.OSNamespace)

	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	if c.commonFlags.Verbose() {
		fmt.Printf("Using UFS service %s\n", e.UnifiedFleetService)
	}

	client := rpc.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})

	lse, err := client.GetMachineLSE(ctx, &rpc.GetMachineLSERequest{
		Name: util.AddPrefix(util.MachineLSECollection, c.dutName),
	})
	if err != nil {
		return err
	}
	if err := utils.IsDUT(lse); err != nil {
		return errors.Annotate(err, "DUT name is not a chromeOS machine").Err()
	}

	var (
		peripherals = lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
		currentBTPs = peripherals.GetBluetoothPeers()
	)

	nb, err := c.newBTPs(currentBTPs)
	if err != nil {
		return err
	}
	if c.commonFlags.Verbose() {
		fmt.Println("New Bluetooth peers list", nb)
	}

	peripherals.BluetoothPeers = nb
	// TODO(b/226024082): Currently field masks are implemented in a limited way. Subsequent update
	// on UFS could add field masks for BTPs and then they could be included here.
	_, err = client.UpdateMachineLSE(ctx, &rpc.UpdateMachineLSERequest{MachineLSE: lse})
	return err
}

// newBTPs returns a new list of BTPs based on the action specified in c and current list.
func (c *manageBTPsCmd) newBTPs(current []*lab.BluetoothPeer) ([]*lab.BluetoothPeer, error) {
	switch c.mode {
	case actionAdd:
		return c.addBTPs(current)
	case actionReplace:
		if c.commonFlags.Verbose() {
			fmt.Println("Removing", current)
		}
		return c.passedBTPs(), nil
	case actionDelete:
		return c.deleteBTPs(current)
	default:
		return nil, errors.Reason("unknown action %d", c.mode).Err()
	}
}

// passedBTPs returns a list of BTPs passed on the CLI.
func (c *manageBTPsCmd) passedBTPs() []*lab.BluetoothPeer {
	var ret []*lab.BluetoothPeer
	for _, h := range c.hostnames {
		ret = append(ret, createBTP(h))
	}
	return ret
}

// addBTPs takes the current list of BTPs and returns a new list with BTPs specified in c added.
// It returns an error if a duplicate is specified.
func (c *manageBTPsCmd) addBTPs(current []*lab.BluetoothPeer) ([]*lab.BluetoothPeer, error) {
	var ret []*lab.BluetoothPeer
	for _, btp := range current {
		h := btp.GetRaspberryPi().GetHostname()
		if c.hostsMap[h] {
			return nil, errors.Reason("BTP host %s already exists", h).Err()
		}
		ret = append(ret, btp)
	}
	ret = append(ret, c.passedBTPs()...)
	return ret, nil
}

// deleteBTPs returns a new slice of BTPs by removing those specified in c from current.
// It returns an error if a non-existent BTP is attempted to be removed.
func (c *manageBTPsCmd) deleteBTPs(current []*lab.BluetoothPeer) ([]*lab.BluetoothPeer, error) {
	currentMap := make(map[string]*lab.BluetoothPeer)
	for _, btp := range current {
		currentMap[btp.GetRaspberryPi().GetHostname()] = btp
	}

	for _, h := range c.hostnames {
		if _, ok := currentMap[h]; !ok {
			return nil, errors.Reason("BTP host %s does not exist", h).Err()
		}
		delete(currentMap, h)
	}

	var ret []*lab.BluetoothPeer
	for _, btp := range currentMap {
		ret = append(ret, btp)
	}
	return ret, nil
}

const (
	errDUTMissing        = "'-dut' is required"
	errNoHostname        = "at least one '-hostname' is required"
	errDuplicateHostname = "duplicate hostname specified"
	errEmptyHostname     = "empty hostname"
)

// cleanAndValidateFlags returns an error with the result of all validations. It strips whitespaces
// around hostnames and removes empty ones.
func (c *manageBTPsCmd) cleanAndValidateFlags() error {
	var errStrs []string
	if len(c.dutName) == 0 {
		errStrs = append(errStrs, errDUTMissing)
	}

	if c.hostsMap == nil {
		c.hostsMap = map[string]bool{}
	}
	var hostnames []string
	for _, h := range c.hostnames {
		h = strings.TrimSpace(h)

		// Empty hostname
		if len(h) == 0 {
			if c.commonFlags.Verbose() {
				fmt.Println("Empty hostname specified")
			}
			errStrs = append(errStrs, errEmptyHostname)
			continue
		}

		// Duplicate hostname
		if c.hostsMap[h] {
			if c.commonFlags.Verbose() {
				fmt.Println("Duplicate hostname specified:", h)
			}
			errStrs = append(errStrs, fmt.Sprintf("%s: %s", errDuplicateHostname, h))
			continue
		}
		c.hostsMap[h] = true
		hostnames = append(hostnames, h)
	}
	c.hostnames = hostnames
	if len(c.hostnames) == 0 {
		errStrs = append(errStrs, errNoHostname)
	}

	if len(errStrs) == 0 {
		return nil
	}

	return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf("Wrong usage!!\n%s", strings.Join(errStrs, "\n")))
}

// createBTP creates a *lab.BluetoothPeer object with initial working state.
func createBTP(hostname string) *lab.BluetoothPeer {
	return &lab.BluetoothPeer{
		Device: &lab.BluetoothPeer_RaspberryPi{
			RaspberryPi: &lab.RaspberryPi{
				Hostname: hostname,
				State:    lab.PeripheralState_UNKNOWN,
			},
		},
	}
}
