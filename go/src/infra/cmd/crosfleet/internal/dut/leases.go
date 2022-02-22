// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"fmt"
	"infra/cmd/crosfleet/internal/buildbucket"
	"infra/cmd/crosfleet/internal/common"
	dutinfopb "infra/cmd/crosfleet/internal/proto"
	"infra/cmd/crosfleet/internal/site"
	"infra/cmd/crosfleet/internal/ufs"
	"infra/cmdsupport/cmdlib"
	"strings"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/cli"
	"google.golang.org/genproto/protobuf/field_mask"
)

const leasesCmd = "leases"

var leases = &subcommands.Command{
	UsageLine: fmt.Sprintf("%s [FLAGS...]", leasesCmd),
	ShortDesc: "print information on the current user's leases",
	LongDesc: `Print information on the current user's leases.

This command's behavior is subject to change without notice.
Do not build automation around this subcommand.`,
	CommandRun: func() subcommands.CommandRun {
		c := &leasesRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.printer.Register(&c.Flags)
		c.Flags.BoolVar(&c.full, "full", false, "Output full DUT/servo info for each lease.")
		c.Flags.BoolVar(&c.includeFinished, "include-finished", false, "Include finished builds.")
		c.Flags.Int64Var(&c.hoursBack, "hours-back", maxLeaseLengthMinutes/60, `Max time since lease finished. Only applies if including finished leases in the search
via the -include-finished flag.`)
		return c
	},
}

type leasesRun struct {
	subcommands.CommandRunBase
	full            bool
	includeFinished bool
	hoursBack       int64
	authFlags       authcli.Flags
	envFlags        common.EnvFlags
	printer         common.CLIPrinter
}

func (c *leasesRun) Run(a subcommands.Application, _ []string, env subcommands.Env) int {
	if err := c.innerRun(a, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *leasesRun) innerRun(a subcommands.Application, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ufsClient, err := ufs.NewUFSClient(ctx, c.envFlags.Env().UFSService, &c.authFlags)
	if err != nil {
		return err
	}
	currentUser, err := common.GetUserEmail(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	leasesBBClient, err := buildbucket.NewClient(ctx, c.envFlags.Env().DUTLeaserBuilder, c.envFlags.Env().BuildbucketService, c.authFlags)
	if err != nil {
		return err
	}
	fieldMask := &field_mask.FieldMask{Paths: []string{
		"builds.*.created_by",
		"builds.*.id",
		"builds.*.create_time",
		"builds.*.start_time",
		"builds.*.status",
		"builds.*.input",
		"builds.*.infra",
		"builds.*.tags",
	}}
	scheduledLeases, err := leasesBBClient.GetAllBuildsByUser(ctx, currentUser, &buildbucketpb.SearchBuildsRequest{
		Predicate: &buildbucketpb.BuildPredicate{
			Status: buildbucketpb.Status_SCHEDULED,
		},
		Fields: fieldMask,
	})
	if err != nil {
		return err
	}
	startedLeases, err := leasesBBClient.GetAllBuildsByUser(ctx, currentUser, &buildbucketpb.SearchBuildsRequest{
		Predicate: &buildbucketpb.BuildPredicate{
			Status: buildbucketpb.Status_STARTED,
		},
		Fields: fieldMask,
	})
	if err != nil {
		return err
	}
	// Display leases in order scheduled -> started -> cancelled.
	sortedLeaseBuilds := append(scheduledLeases, startedLeases...)

	if c.includeFinished {
		finishedLeases, err := leasesBBClient.GetAllBuildsByUser(ctx, currentUser, &buildbucketpb.SearchBuildsRequest{
			Predicate: &buildbucketpb.BuildPredicate{
				Status: buildbucketpb.Status_ENDED_MASK,
				CreateTime: &buildbucketpb.TimeRange{
					StartTime: common.OffsetTimestamp(-60 * c.hoursBack),
				},
			},
			Fields: fieldMask,
		})
		if err != nil {
			return err
		}
		sortedLeaseBuilds = append(sortedLeaseBuilds, finishedLeases...)
	}

	var leaseInfoList dutinfopb.LeaseInfoList
	for _, build := range sortedLeaseBuilds {
		leaseInfo := &dutinfopb.LeaseInfo{Build: build}
		dutHostname := buildbucket.FindDimValInFinalDims("dut_name", build)
		if dutHostname != "" {
			if c.full {
				leaseInfo.DUT, err = getDutInfo(ctx, ufsClient, dutHostname)
				if err != nil {
					return err
				}
			} else {
				leaseInfo.DUT = &dutinfopb.DUTInfo{Hostname: dutHostname}
			}
		}
		// If outputting the command as JSON, collect all lease info in a proto
		// message first, then print together as one JSON object.
		// Otherwise, just print each separately from this loop.
		leaseInfoList.Leases = append(leaseInfoList.Leases, leaseInfo)
		c.printer.WriteTextStdout("%s\n", leaseInfoAsBashVariables(leaseInfo, leasesBBClient))
	}
	c.printer.WriteJSONStdout(&leaseInfoList)

	return nil
}

// leaseInfoAsBashVariables returns a pretty-printed string containing info
// about the given lease formatted as bash variables. Only the variables that
// are found in the lease info proto message are printed.
func leaseInfoAsBashVariables(info *dutinfopb.LeaseInfo, leasesBBClient *buildbucket.Client) string {
	var bashVars []string

	build := info.GetBuild()
	if build != nil {
		bashVars = append(bashVars,
			fmt.Sprintf("LEASE_TASK=%s\nSTATUS=%s\nMINS_REMAINING=%d",
				leasesBBClient.BuildURL(build.GetId()),
				build.GetStatus(),
				getRemainingMins(build)))
	}

	dut := info.GetDUT()
	if dut != nil {
		bashVars = append(bashVars, dutInfoAsBashVariables(dut))
	}

	return strings.Join(bashVars, "\n")
}

// getRemainingMins gets the remaining minutes on a lease from a given
// dut_leaser Buildbucket build.
func getRemainingMins(build *buildbucketpb.Build) int64 {
	inputProps := build.GetInput().GetProperties().GetFields()
	leaseLengthMins := inputProps["lease_length_minutes"].GetNumberValue()
	status := build.GetStatus()
	switch status {
	case buildbucketpb.Status_SCHEDULED:
		// Lease hasn't started; full lease length remains.
		return int64(leaseLengthMins)
	case buildbucketpb.Status_STARTED:
		// Lease has started; subtract elapsed time from lease length.
		minsElapsed := time.Now().Sub(build.GetStartTime().AsTime()).Minutes()
		return int64(leaseLengthMins - minsElapsed)
	default:
		// Lease is finished; no time remains.
		return 0
	}
}
