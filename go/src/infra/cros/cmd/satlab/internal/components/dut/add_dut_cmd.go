// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"fmt"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/errors"

	"infra/cmdsupport/cmdlib"

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
		// Manual_tags must be key:value form.
		c.deployTags = []string{"satlab:true"}
		// TODO(gregorynisbet): Consider skipping actions for satlab by default.
		c.deployActions = defaultDeployTaskActions
		c.assetType = "dut"

		c.Flags.StringVar(&c.address, "address", "", "IP address of host")
		c.Flags.BoolVar(&c.skipDNS, "skip-dns", false, "whether to skip updating the DNS")
		c.Flags.BoolVar(
			&c.fullDeploy,
			"full-deploy",
			false,
			"whether to run full deployment workflow(include OS and firmware install)")
		registerAddShivasFlags(c)
		return c
	},
}

// AddDUT contains the arguments for "satlab add dut ...". It also contains additional
// qualified arguments that are the result of adding the satlab prefix to "raw" arguments.
type addDUT struct {
	shivasAddDUT
	// AssetType is the type of the asset, it always has a value of "dut".
	assetType string
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
	// FullDeploy is run complete deployment task which includes OS and Firmware installs
	fullDeploy bool
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
func (c *addDUT) innerRun(a subcommands.Application, args []string, env subcommands.Env) (err error) {
	// This function has a single defer block that inspects the return value err to see if it
	// is nil. This defer block does *not* set the err back to nil if it succeeds in cleaning up
	// the dut_hosts file. Instead, it creates a multierror with whatever errors it encountered.
	//
	// If we're going to add multiple defer blocks, a different strategy is needed to make sure that
	// they compose in the correct way.
	dockerHostBoxIdentifier, err := getDockerHostBoxIdentifier(c.commonFlags)
	if err != nil {
		return errors.Annotate(err, "add dut").Err()
	}

	// Set Satlab identifier as default pool if not  given".
	if len(c.pools) == 0 {
		defaultPool := fmt.Sprintf("%s-%s", site.Satlab, dockerHostBoxIdentifier)
		c.pools = append(c.pools, defaultPool)
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
		c.qualifiedServo = site.MaybePrepend(
			site.Satlab,
			dockerHostBoxIdentifier,
			fmt.Sprintf(
				"%s-%s",
				c.hostname,
				"docker_servod:9999",
			),
		)
		if c.servoDockerContainerName == "" {
			c.servoDockerContainerName = site.MaybePrepend(
				site.Satlab,
				dockerHostBoxIdentifier,
				fmt.Sprintf(
					"%s-%s",
					c.hostname,
					"docker_servod",
				),
			)
		}

	} else {
		c.qualifiedServo = site.MaybePrepend(site.Satlab, dockerHostBoxIdentifier, c.servo)
	}

	if c.zone == "" {
		c.zone = site.DefaultZone
	}

	// Update the DNS entry first. This step must run before we deploy the DUT.
	// This step can occur in any order with respect to ensuring the existence of the rack or
	// the asset.
	if !c.skipDNS {
		content, updateErr := dns.UpdateRecord(
			c.qualifiedHostname,
			c.address,
		)
		if updateErr != nil {
			return errors.Annotate(updateErr, "add dut").Err()
		}
		// Write the content back if we fail at a later step for any reason.
		defer (func() {
			// Err refers to the error for the function as a whole.
			// If it's non-nil, then a later step has failed and we need
			// to clean up after ourselves.
			if content == "" {
				// If the content is empty, do nothing because we either failed to
				// copy the contents of the file, or the file was empty originally.
				//
				// In either case, restoring the old contents could potentially lose
				// information.
				//
				// Do not modify the error value.
				if err != nil {
					fmt.Fprintf(os.Stderr, "original DNS entry was empty.\n")
				} else {
					fmt.Fprintf(os.Stderr, "original DNS entry was empty. Skipping restoration\n")
				}
			} else if err != nil {
				fmt.Fprintf(os.Stderr, "Restoring DNS content after failed step\n")
				dnsErr := dns.SetDNSFileContent(content)
				fmt.Fprintf(os.Stderr, "Restarting DNSMasq after failed step\n")
				reloadErr := dns.ForceReloadDNSMasqProcess()
				err = errors.NewMultiError(err, dnsErr, reloadErr)
			}
		})()
	}

	// For Satlab,  default we skip certain deployment task such as
	// downloading image, installing OS and firmware".
	if !c.fullDeploy {
		c.deploySkipDownloadImage = true
		c.deploySkipInstallOS = true
		c.deploySkipInstallFirmware = true
		c.deploySkipRecoveryMode = true
	} else {
		c.deployActions = append(c.deployActions, "verify-recovery-mode")
	}

	if err := (&shivas.Rack{
		Name:      c.qualifiedRack,
		Namespace: c.envFlags.Namespace,
		Zone:      c.zone,
	}).CheckAndAdd(); err != nil {
		return errors.Annotate(err, "add dut").Err()
	}

	if err := (&shivas.Asset{
		Asset:     c.asset,
		Rack:      c.qualifiedRack,
		Zone:      c.zone,
		Model:     c.model,
		Board:     c.board,
		Namespace: c.envFlags.Namespace,
		Type:      c.assetType,
	}).CheckAndAdd(); err != nil {
		return errors.Annotate(err, "add dut").Err()
	}

	if err := (&shivas.DUT{
		Namespace:  c.envFlags.Namespace,
		Zone:       c.zone,
		Name:       c.qualifiedHostname,
		Rack:       c.qualifiedRack,
		Servo:      c.qualifiedServo,
		ShivasArgs: makeAddShivasFlags(c),
	}).CheckAndAdd(); err != nil {
		return errors.Annotate(err, "add dut").Err()
	}

	return nil
}
