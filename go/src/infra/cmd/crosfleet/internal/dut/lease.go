// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"flag"
	"fmt"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"infra/cmd/crosfleet/internal/buildbucket"
	"infra/cmd/crosfleet/internal/common"
	"infra/cmd/crosfleet/internal/flagx"
	"infra/cmd/crosfleet/internal/site"
	"infra/cmdsupport/cmdlib"
)

const (
	dayInMinutes = 24 * 60
	// Buildbucket priority for dut_leaser builds.
	dutLeaserBuildPriority = 15
	// leaseCmdName is the name of the `crosfleet dut lease` command.
	leaseCmdName = "lease"
	// DUT pool available for leasing from.
	leasesPool               = "DUT_POOL_QUOTA"
	maxLeaseReasonCharacters = 30
	// maxLeasesPerModel is the max number of leases allowed at a time for a
	// given model.
	maxLeasesPerModel = 1
)

var lease = &subcommands.Command{
	UsageLine: fmt.Sprintf("%s {-model MODEL/-host HOST}", leaseCmdName),
	ShortDesc: "lease DUT for debugging",
	LongDesc: `Lease DUT for debugging.

DUTs can be leased by model or by individual DUT hostname.
Leasing by model is faster, since it reserves the first available DUT of the given model. 

This command's behavior is subject to change without notice.
Do not build automation around this subcommand.`,
	CommandRun: func() subcommands.CommandRun {
		c := &leaseRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.leaseFlags.register(&c.Flags)
		return c
	},
}

type leaseRun struct {
	subcommands.CommandRunBase
	leaseFlags
	authFlags authcli.Flags
	envFlags  common.EnvFlags
}

func (c *leaseRun) Run(a subcommands.Application, _ []string, env subcommands.Env) int {
	if err := c.innerRun(a, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	fmt.Fprintf(a.GetOut(), "In real life this would start a lease.\n")
	return 0
}

func (c *leaseRun) innerRun(a subcommands.Application, env subcommands.Env) error {
	if err := c.leaseFlags.validate(&c.Flags); err != nil {
		return err
	}
	buildProps := map[string]interface{}{
		"lease_length_minutes": c.durationMins,
	}
	botDims, buildTags := c.botDimsAndBuildTags()

	ctx := cli.GetContext(a, c, env)
	bbClient, err := buildbucket.NewClient(ctx, c.envFlags.Env().DUTLeaserBuilderInfo, c.authFlags)
	if err != nil {
		return err
	}
	buildID, err := bbClient.ScheduleBuild(ctx, buildProps, botDims, buildTags, dutLeaserBuildPriority)
	fmt.Fprintf(a.GetOut(), "Leasing DUT at %s\n", bbClient.BuildURL(buildID))
	return nil
}

// botDimsAndBuildTags constructs bot dimensions and Buildbucket build tags for
// a dut_leaser build.
func (c *leaseFlags) botDimsAndBuildTags() (dims, tags map[string]string) {
	dims = map[string]string{}
	tags = map[string]string{}

	// Add user-added dimensions to both bot dimensions and build tags.
	for key, val := range c.addedDims {
		dims[key] = val
		tags[key] = val
	}

	if c.model != "" {
		dims["label-model"] = c.model
		dims["label-pool"] = leasesPool
		// Only look for DUTs that are ready for a new task, i.e. not broken.
		dims["dut_state"] = "ready"

		tags["model"] = c.model
		tags["lease-by"] = "model"
	} else if c.host != "" {
		dims["id"] = c.host

		tags["id"] = c.host
		tags["lease-by"] = "host"
	}

	tags["crosfleet-tool"] = leaseCmdName
	tags["lease-reason"] = c.reason
	tags["qs-account"] = "leases"

	return
}

// leaseFlags contains parameters for the "dut lease" subcommand.
type leaseFlags struct {
	durationMins int64
	reason       string
	host         string
	model        string
	addedDims    map[string]string
}

// Registers lease-specific flags.
func (c *leaseFlags) register(f *flag.FlagSet) {
	f.Int64Var(&c.durationMins, "minutes", 60, "Duration of lease in minutes.")
	f.StringVar(&c.reason, "reason", "", fmt.Sprintf("Optional reason for leasing (limit %d characters).", maxLeaseReasonCharacters))
	f.StringVar(&c.model, "model", "", "Model of DUT to lease. If leasing by model, the first available DUT of the given model will be leased.")
	f.StringVar(&c.host, "host", "", `Hostname of an individual DUT to lease. If leasing by hostname and the host DUT is running another task,
the lease won't start until that task completes.`)
	f.Var(flagx.KeyVals(&c.addedDims), "dim", "Additional DUT dimension in format key=val or key:val; may be specified multiple times.")
	f.Var(flagx.KeyVals(&c.addedDims), "dims", "Comma-separated additional DUT dimensions in same format as -dim.")
}

func (c *leaseFlags) validate(f *flag.FlagSet) error {
	var errors []string
	if c.model != "" && c.host != "" {
		errors = append(errors, "model and host cannot both be specified")
	}
	if c.model == "" && c.host == "" {
		errors = append(errors, "missing either model or host")
	}
	if c.durationMins <= 0 {
		errors = append(errors, "duration should be greater than 0")
	}
	if c.durationMins > dayInMinutes {
		errors = append(errors, fmt.Sprintf("duration cannot exceed 24 hours (%d minutes)", dayInMinutes))
	}
	if len(c.reason) > maxLeaseReasonCharacters {
		errors = append(errors, fmt.Sprintf("reason cannot exceed %d characters", maxLeaseReasonCharacters))
	}

	if len(errors) > 0 {
		return cmdlib.NewUsageError(*f, strings.Join(errors, "\n"))
	}
	return nil
}
