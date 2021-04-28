// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"
	"flag"
	"fmt"
	"infra/cmd/crosfleet/internal/buildbucket"
	"infra/cmd/crosfleet/internal/common"
	"infra/cmd/crosfleet/internal/flagx"
	"infra/cmd/crosfleet/internal/site"
	"infra/cmdsupport/cmdlib"
	"strings"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	swarmingapi "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/cli"
)

const (
	// maxLeaseLengthMinutes is 24 hours in minutes.
	maxLeaseLengthMinutes = 24 * 60
	// Buildbucket priority for dut_leaser builds.
	dutLeaserBuildPriority = 15
	// leaseCmdName is the name of the `crosfleet dut lease` command.
	leaseCmdName = "lease"
	// Default DUT pool available for leasing from.
	defaultLeasesPool        = "DUT_POOL_QUOTA"
	maxLeaseReasonCharacters = 30
)

var lease = &subcommands.Command{
	UsageLine: fmt.Sprintf("%s [FLAGS...]", leaseCmdName),
	ShortDesc: "lease DUT for debugging",
	LongDesc: `Lease DUT for debugging.

DUTs can be leased by Swarming dimensions or by individual DUT hostname.
Leasing by dimensions is fastest, since the first available DUT matching the
requested dimensions is reserved. 'label-board' and 'label-model' dimensions can
be specified via the -board and -model flags, respectively; other Swarming
dimensions can be specified via the freeform -dim/-dims flags.

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
	return 0
}

func (c *leaseRun) innerRun(a subcommands.Application, env subcommands.Env) error {
	if err := c.leaseFlags.validate(&c.Flags); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	swarmingService, err := newSwarmingService(ctx, c.envFlags.Env().SwarmingService, &c.authFlags)
	if err != nil {
		return err
	}
	botDims, buildTags, err := botDimsAndBuildTags(ctx, swarmingService, c.leaseFlags)
	if err != nil {
		return err
	}
	buildProps := map[string]interface{}{
		"lease_length_minutes": c.durationMins,
	}

	leasesBBClient, err := buildbucket.NewClient(ctx, c.envFlags.Env().DUTLeaserBuilder, c.envFlags.Env().BuildbucketService, c.authFlags)
	if err != nil {
		return err
	}
	buildID, err := leasesBBClient.ScheduleBuild(ctx, buildProps, botDims, buildTags, dutLeaserBuildPriority)
	fmt.Fprintf(a.GetErr(), "Requesting %d minute lease at %s\n", c.durationMins, leasesBBClient.BuildURL(buildID))
	if c.exitEarly {
		return nil
	}
	fmt.Fprintf(a.GetErr(), "Waiting to confirm DUT %s request validation and print leased DUT details...\n(To skip this step, pass the -exit-early flag on future DUT %s commands)\n", leaseCmdName, leaseCmdName)
	build, err := leasesBBClient.WaitForBuildStepStart(ctx, buildID, c.leaseStartStepName())
	if err != nil {
		return err
	}
	host := buildbucket.FindDimValInFinalDims("dut_name", build)
	endTime := time.Now().Add(time.Duration(c.durationMins) * time.Minute).Format(time.RFC822)
	fmt.Fprintf(a.GetOut(), "Leased %s until %s\n\n", host, endTime)
	ufsClient, err := newUFSClient(ctx, c.envFlags.Env().UFSService, &c.authFlags)
	if err != nil {
		// Don't fail the command here, since the DUT is already leased.
		fmt.Fprintf(a.GetErr(), "Unable to contact UFS to print DUT info: %v", err)
		return nil
	}
	dutInfo, err := getDutInfo(ctx, ufsClient, host)
	if err != nil {
		// Don't fail the command here, since the DUT is already leased.
		fmt.Fprintf(a.GetErr(), "Unable to print DUT info: %v", err)
		return nil
	}
	fmt.Fprintf(a.GetErr(), "%s\n\n", dutInfoAsBashVariables(dutInfo))
	return nil
}

// botDimsAndBuildTags constructs bot dimensions and Buildbucket build tags for
// a dut_leaser build from the given lease flags and optional bot ID.
func botDimsAndBuildTags(ctx context.Context, swarmingService *swarmingapi.Service, leaseFlags leaseFlags) (dims, tags map[string]string, err error) {
	dims = map[string]string{}
	tags = map[string]string{}
	if leaseFlags.host != "" {
		// Hostname-based lease.
		correctedHostname := correctedHostname(leaseFlags.host)
		id, err := hostnameToBotID(ctx, swarmingService, correctedHostname)
		if err != nil {
			return nil, nil, err
		}
		tags["lease-by"] = "host"
		tags["id"] = id
		dims["id"] = id
	} else {
		// Swarming dimension-based lease.
		dims["dut_state"] = "ready"
		userSpecifiedPool := false
		// Add user-added dimensions to both bot dimensions and build tags.
		for key, val := range leaseFlags.freeformDims {
			if key == "label-pool" {
				userSpecifiedPool = true
			}
			dims[key] = val
			tags[key] = val
		}
		if !userSpecifiedPool {
			dims["label-pool"] = defaultLeasesPool
		}
		if board := leaseFlags.board; board != "" {
			tags["label-board"] = board
			dims["label-board"] = board
		}
		if model := leaseFlags.model; model != "" {
			tags["label-model"] = model
			dims["label-model"] = model
		}
	}
	// Add these metadata tags last to avoid being overwritten by freeform dims.
	tags["crosfleet-tool"] = leaseCmdName
	tags["lease-reason"] = leaseFlags.reason
	tags["qs_account"] = "leases"
	return
}

// leaseFlags contains parameters for the "dut lease" subcommand.
type leaseFlags struct {
	durationMins int64
	reason       string
	host         string
	model        string
	board        string
	freeformDims map[string]string
	exitEarly    bool
}

// Registers lease-specific flags.
func (c *leaseFlags) register(f *flag.FlagSet) {
	f.Int64Var(&c.durationMins, "minutes", 60, "Duration of lease in minutes.")
	f.StringVar(&c.reason, "reason", "", fmt.Sprintf("Optional reason for leasing (limit %d characters).", maxLeaseReasonCharacters))
	f.StringVar(&c.board, "board", "", "'label-board' Swarming dimension to lease DUT by.")
	f.StringVar(&c.model, "model", "", "'label-model' Swarming dimension to lease DUT by.")
	f.StringVar(&c.host, "host", "", `Hostname of an individual DUT to lease. If leasing by hostname instead of other Swarming dimensions,
and the host DUT is running another task, the lease won't start until that task completes.
Mutually exclusive with -board/-model/-dim(s).`)
	f.Var(flagx.KeyVals(&c.freeformDims), "dim", "Freeform Swarming dimension to lease DUT by, in format key=val or key:val; may be specified multiple times.")
	f.Var(flagx.KeyVals(&c.freeformDims), "dims", "Comma-separated Swarming dimensions, in same format as -dim.")
	f.BoolVar(&c.exitEarly, "exit-early", false, `Exit command as soon as lease is scheduled. crosfleet will not notify on lease validation failure,
or print the hostname of the leased DUT.`)
}

func (c *leaseFlags) validate(f *flag.FlagSet) error {
	var errors []string
	if !c.hasEitherHostnameOrSwarmingDims() {
		errors = append(errors, "must specify DUT dimensions (-board/-model/-dim(s)) or DUT hostname (-host), but not both")
	}
	if c.durationMins <= 0 {
		errors = append(errors, "duration should be greater than 0")
	}
	if c.durationMins > maxLeaseLengthMinutes {
		errors = append(errors, fmt.Sprintf("duration cannot exceed %d minutes (%d hours)", maxLeaseLengthMinutes, maxLeaseLengthMinutes/60))
	}
	if len(c.reason) > maxLeaseReasonCharacters {
		errors = append(errors, fmt.Sprintf("reason cannot exceed %d characters", maxLeaseReasonCharacters))
	}

	if len(errors) > 0 {
		return cmdlib.NewUsageError(*f, strings.Join(errors, "\n"))
	}
	return nil
}

// hasOnePrimaryDim verifies that the lease flags contain either a DUT hostname
// or swarming dimensions (via -board/-model/-dim(s)), but not both.
func (c *leaseFlags) hasEitherHostnameOrSwarmingDims() bool {
	hasHostname := c.host != ""
	hasSwarmingDims := c.board != "" || c.model != "" || len(c.freeformDims) > 0
	return hasHostname != hasSwarmingDims
}

func (c *leaseRun) leaseStartStepName() string {
	hours := c.durationMins / 60
	mins := c.durationMins % 60
	return fmt.Sprintf("lease DUT for %d hr %d min", hours, mins)
}
