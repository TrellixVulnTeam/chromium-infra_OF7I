// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"fmt"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/utils"
	"infra/cros/cmd/satlab/internal/site"
	"infra/libs/swarming"
)

// Flagmap is a map from the name of a flag to its value(s).
type flagmap = map[string][]string

// MakeShivasFlags takes an add DUT command and serializes its flags in such
// a way that they will parse to same values.
func makeAddShivasFlags(c *addDUT) flagmap {
	out := make(flagmap)

	// These other flags are inherited from shivas.
	if c.newSpecsFile != "" {
		// Do nothing.
		// This flag is intentionally unsupported.
		// We tweak the names of fields therefore we cannot deploy
		// using a spec file.
	}
	if c.zone != "" {
		out["zone"] = []string{c.zone}
	}
	if c.rack != "" {
		out["rack"] = []string{c.rack}
	}
	if c.hostname != "" {
		out["name"] = []string{c.hostname}
	}
	if c.asset != "" {
		out["asset"] = []string{c.asset}
	}
	if c.servo != "" {
		out["servo"] = []string{c.servo}
	}
	if c.servoSerial != "" {
		out["servo-serial"] = []string{c.servoSerial}
	}
	if c.servoSetupType != "" {
		out["servo-setup"] = []string{c.servoSetupType}
	}
	if len(c.pools) != 0 {
		out["pools"] = []string{strings.Join(c.pools, ",")}
	}
	if len(c.licenseTypes) != 0 {
		out["licensetype"] = []string{strings.Join(c.licenseTypes, ",")}
	}
	if c.rpm != "" {
		out["rpm"] = []string{c.rpm}
	}
	if c.rpmOutlet != "" {
		out["rpm-outlet"] = []string{c.rpmOutlet}
	}
	if c.deployTaskTimeout != 0 {
		out["deploy-timeout"] = []string{fmt.Sprintf("%d", c.deployTaskTimeout)}
	}
	if c.ignoreUFS {
		// This flag is unsupported.
	}
	if len(c.deployTags) != 0 {
		out["deploy-tags"] = []string{strings.Join(c.deployTags, ",")}
	}
	if c.deploySkipDownloadImage {
		out["deploy-skip-download-image"] = []string{}
	}
	if c.deploySkipInstallFirmware {
		out["deploy-skip-install-firmware"] = []string{}
	}
	if c.deploySkipInstallOS {
		out["deploy-skip-install-os"] = []string{}
	}
	if len(c.tags) != 0 {
		out["tags"] = []string{strings.Join(c.tags, ",")}
	}
	if c.state != "" {
		// This flag is unsupported.
	}
	if c.description != "" {
		out["desc"] = []string{c.description}
	}
	if len(c.chameleons) != 0 {
		out["chameleons"] = []string{strings.Join(c.chameleons, ",")}
	}
	if len(c.cameras) != 0 {
		out["cameras"] = []string{strings.Join(c.cameras, ",")}
	}
	if len(c.cables) != 0 {
		out["cables"] = []string{strings.Join(c.cables, ",")}
	}
	if c.antennaConnection != "" {
		out["antennaconnection"] = []string{c.antennaConnection}
	}
	if c.router != "" {
		out["router"] = []string{c.router}
	}
	if c.facing != "" {
		out["facing"] = []string{c.facing}
	}
	if c.light != "" {
		out["light"] = []string{c.light}
	}
	if c.carrier != "" {
		out["carrier"] = []string{c.carrier}
	}
	if c.audioBoard {
		out["audioboard"] = []string{}
	}
	if c.audioBox {
		out["audiobox"] = []string{}
	}
	if c.atrus {
		out["atrus"] = []string{}
	}
	if c.wifiCell {
		out["wificell"] = []string{}
	}
	if c.touchMimo {
		out["touchmimo"] = []string{}
	}
	if c.cameraBox {
		out["camerabox"] = []string{}
	}
	if c.chaos {
		out["chaos"] = []string{}
	}
	if c.audioCable {
		out["audiocable"] = []string{}
	}
	if c.smartUSBHub {
		out["smartusbhub"] = []string{}
	}
	if c.model != "" {
		out["model"] = []string{}
	}
	if c.board != "" {
		out["board"] = []string{}
	}
	if c.envFlags.Namespace != "" {
		out["namespace"] = []string{c.envFlags.Namespace}
	}
	return out
}

// ShivasAddDUT contains all the commands for "satlab add dut" inherited from shivas.
//
// Keep this up to date with infra/cmd/shivas/ufs/subcmds/dut/add_dut.go
type shivasAddDUT struct {
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

// DefaultDeployTaskActions are the default actoins run at deploy time.
// TODO(gregorynisbet): this about which actions make sense for satlab.
var defaultDeployTaskActions = []string{"servo-verification", "update-label", "verify-recovery-mode", "run-pre-deploy-verification"}

// Register flags inherited from shivas in place in the add DUT command.
// Keep this up to date with infra/cmd/shivas/ufs/subcmds/dut/add_dut.go
func registerAddShivasFlags(c *addDUT) {
	c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
	c.envFlags.Register(&c.Flags)
	c.commonFlags.Register(&c.Flags)

	c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.DUTRegistrationFileText)

	// Asset location fields
	c.Flags.StringVar(&c.zone, "zone", site.DefaultZone, "Zone that the asset is in. "+cmdhelp.ZoneFilterHelpText)
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
}
