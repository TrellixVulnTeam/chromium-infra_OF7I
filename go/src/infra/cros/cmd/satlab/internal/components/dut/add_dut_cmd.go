// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"fmt"
	"os"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/errors"

	"infra/cmdsupport/cmdlib"

	"infra/cros/cmd/satlab/internal/commands"
	"infra/cros/cmd/satlab/internal/commands/dns"
	"infra/cros/cmd/satlab/internal/components/dut/shivas"
	"infra/cros/cmd/satlab/internal/site"
)

// AddDUTCmd is the command that deploys a Satlab DUT.
var AddDUTCmd = &subcommands.Command{
	UsageLine: "dut [options ...]",
	ShortDesc: "Deploy a Satlab DUT",
	CommandRun: func() subcommands.CommandRun {

		// keep this up to date with infra/cmd/shivas/ufs/subcmds/dut/add_dut.go
		c := &addDUT{}
		c.pools = []string{}
		c.chameleons = []string{}
		c.cameras = []string{}
		c.cables = []string{}
		// TODO(gregorynisbet): Add more info here.
		c.deployTags = []string{"satlab"}
		// TODO(gregorynisbet): Consider skipping actions for satlab by default.
		c.deployActions = defaultDeployTaskActions

		c.Flags.StringVar(&c.address, "address", "", "IP address of host")
		c.Flags.BoolVar(&c.skipDNS, "skip-dns", false, "whether to skip updating the DNS")
		registerAddShivasFlags(c)
		return c
	},
}

// AddDUT contains the arguments for "satlab add dut ...". It also contains additional
// qualified arguments that are the result of adding the satlab prefix to "raw" arguments.
type addDUT struct {
	shivasAddDUT
	// Satlab-specific fields, if any exist, go here.
	// Address is the IP adderss of the DUT.
	address string
	// SkipDNS controls whether to modify the /etc/dut_hosts/hosts file on the dns container.
	skipDNS bool
	// QualifiedHostname is the hostname with the satlab ID prepended.
	qualifiedHostname string
	// QualifiedServo is the servo with the satlab ID prepended.
	qualifiedServo string
	// QualifiedRack is the rack with the satlab ID prepended.
	qualifiedRack string
}

// Run adds a DUT and returns an exit status.
func (c *addDUT) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

// InnerRun is the implementation of run.
func (c *addDUT) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	dockerHostBoxIdentifier := strings.ToLower(c.commonFlags.SatlabID)
	if dockerHostBoxIdentifier == "" {
		var err error
		dockerHostBoxIdentifier, err = commands.GetDockerHostBoxIdentifier()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to determine -satlab prefix, use %s to pass explicitly\n", c.commonFlags.SatlabID)
			return errors.Annotate(err, "add dut").Err()
		}
	}

	// The qualified name of a rack if no information is given is "satlab-...-rack".
	if c.rack == "" {
		c.rack = "rack"
	}

	c.qualifiedHostname = site.MaybePrepend(site.Satlab, dockerHostBoxIdentifier, c.hostname)
	c.qualifiedRack = site.MaybePrepend(site.Satlab, dockerHostBoxIdentifier, c.rack)
	if c.servo == "" {
		// If no servo configuration is provided, use
		// the docker_servod configuration
		// TODO(gregorynisbet): Add support for
		// -servo-docker flag once shivas supports it.
		c.qualifiedServo = site.MaybePrepend(
			site.Satlab,
			dockerHostBoxIdentifier,
			fmt.Sprintf(
				"%s-%s",
				c.hostname,
				"docker_servod:9999",
			),
		)
	} else {
		c.qualifiedServo = site.MaybePrepend(site.Satlab, dockerHostBoxIdentifier, c.servo)
	}

	if c.zone == "" {
		c.zone = site.DefaultZone
	}

	if err := (&shivas.Rack{
		Name:      c.qualifiedRack,
		Namespace: c.envFlags.Namespace,
		Zone:      c.zone,
	}).CheckAndUpdate(); err != nil {
		return errors.Annotate(err, "add dut").Err()
	}

	if err := (&shivas.Asset{
		Asset:     c.asset,
		Rack:      c.qualifiedRack,
		Zone:      c.zone,
		Model:     c.model,
		Board:     c.board,
		Namespace: c.envFlags.Namespace,
	}).CheckAndUpdate(); err != nil {
		return errors.Annotate(err, "add dut").Err()
	}

	if err := (&shivas.DUT{
		Namespace:  c.envFlags.Namespace,
		Zone:       c.zone,
		Name:       c.qualifiedHostname,
		Servo:      c.qualifiedServo,
		ShivasArgs: makeAddShivasFlags(c),
	}).CheckAndUpdate(); err != nil {
		return errors.Annotate(err, "add dut").Err()
	}

	if !c.skipDNS {
		if err := dns.UpdateRecord(
			c.qualifiedHostname,
			c.address,
		); err != nil {
			return errors.Annotate(err, "add dut").Err()
		}
	}

	return nil
}
