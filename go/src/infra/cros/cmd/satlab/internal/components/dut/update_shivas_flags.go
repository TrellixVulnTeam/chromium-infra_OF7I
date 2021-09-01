// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/utils"
	"infra/cros/cmd/satlab/internal/site"
	"infra/libs/swarming"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
)

// MakeUpdateShivasFlags serializes the command line arguments of updateDUT into a flagmap
// so that it can be used to call shivas directly.
func makeUpdateShivasFlags(c *updateDUT) flagmap {
	panic("not implemented")
}

// ShivasUpdateDUT is a command that contains the arguments that "shivas update" understands.
type shivasUpdateDUT struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	// DUT specification inputs.
	newSpecsFile             string
	hostname                 string
	machine                  string
	servo                    string
	servoSerial              string
	servoSetupType           string
	servoFwChannel           string
	servoDockerContainerName string
	pools                    []string
	licenseTypes             []string
	licenseIds               []string
	rpm                      string
	rpmOutlet                string
	deploymentTicket         string
	tags                     []string
	description              string

	// Deploy task inputs.
	forceDeploy          bool
	deployTaskTimeout    int64
	deployTags           []string
	forceDownloadImage   bool
	forceInstallOS       bool
	forceInstallFirmware bool
	forceUpdateLabels    bool

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

	// For use in determining if a flag is set
	flagInputs map[string]bool
}

// RegisterUpdateShivasFlags registers the flags inherited from shivas.
func registerUpdateShivasFlags(c *updateDUT) {
	c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
	c.envFlags.Register(&c.Flags)
	c.commonFlags.Register(&c.Flags)

	c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.DUTUpdateFileText)

	c.Flags.StringVar(&c.hostname, "name", "", "hostname of the DUT.")
	c.Flags.StringVar(&c.machine, "asset", "", "asset tag of the DUT.")
	c.Flags.StringVar(&c.servo, "servo", "", "servo hostname and port as hostname:port. Clearing this field will delete the servo in DUT. "+cmdhelp.ClearFieldHelpText)
	c.Flags.StringVar(&c.servoSerial, "servo-serial", "", "serial number for the servo.")
	c.Flags.StringVar(&c.servoSetupType, "servo-setup", "", "servo setup type. Allowed values are "+cmdhelp.ServoSetupTypeAllowedValuesString()+".")
	c.Flags.StringVar(&c.servoFwChannel, "servo-fw-channel", "", "servo firmware channel. Allowed values are "+cmdhelp.ServoFwChannelAllowedValuesString()+".")
	c.Flags.StringVar(&c.servoDockerContainerName, "servod-docker", "", "servo docker container name. Required if servod is running in docker.")
	c.Flags.Var(utils.CSVString(&c.pools), "pools", "comma seperated pools. These will be appended to existing pools. "+cmdhelp.ClearFieldHelpText)
	c.Flags.Var(utils.CSVString(&c.licenseTypes), "licensetype", cmdhelp.LicenseTypeHelpText)
	c.Flags.Var(utils.CSVString(&c.licenseIds), "licenseid", "the name of the license type. Can specify multiple comma separated values. "+cmdhelp.ClearFieldHelpText)
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

	// ACS DUT fields
	c.Flags.Var(utils.CSVString(&c.chameleons), "chameleons", cmdhelp.ChameleonTypeHelpText+". "+cmdhelp.ClearFieldHelpText)
	c.Flags.Var(utils.CSVString(&c.cameras), "cameras", cmdhelp.CameraTypeHelpText+". "+cmdhelp.ClearFieldHelpText)
	c.Flags.Var(utils.CSVString(&c.cables), "cables", cmdhelp.CableTypeHelpText+". "+cmdhelp.ClearFieldHelpText)
	c.Flags.StringVar(&c.antennaConnection, "antennaconnection", "", cmdhelp.AntennaConnectionHelpText)
	c.Flags.StringVar(&c.router, "router", "", cmdhelp.RouterHelpText)
	c.Flags.StringVar(&c.facing, "facing", "", cmdhelp.FacingHelpText)
	c.Flags.StringVar(&c.light, "light", "", cmdhelp.LightHelpText)
	c.Flags.StringVar(&c.carrier, "carrier", "", "name of the carrier."+". "+cmdhelp.ClearFieldHelpText)
	c.Flags.BoolVar(&c.audioBoard, "audioboard", false, "adding this flag will specify if audioboard is present")
	c.Flags.BoolVar(&c.audioBox, "audiobox", false, "adding this flag will specify if audiobox is present")
	c.Flags.BoolVar(&c.atrus, "atrus", false, "adding this flag will specify if atrus is present")
	c.Flags.BoolVar(&c.wifiCell, "wificell", false, "adding this flag will specify if wificell is present")
	c.Flags.BoolVar(&c.touchMimo, "touchmimo", false, "adding this flag will specify if touchmimo is present")
	c.Flags.BoolVar(&c.cameraBox, "camerabox", false, "adding this flag will specify if camerabox is present")
	c.Flags.BoolVar(&c.chaos, "chaos", false, "adding this flag will specify if chaos is present")
	c.Flags.BoolVar(&c.audioCable, "audiocable", false, "adding this flag will specify if audiocable is present")
	c.Flags.BoolVar(&c.smartUSBHub, "smartusbhub", false, "adding this flag will specify if smartusbhub is present")
}
