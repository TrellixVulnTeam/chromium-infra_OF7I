// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"fmt"
	"infra/cmd/skylab/internal/bb"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	buildbucket_pb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/flagx"
	"infra/cmd/skylab/internal/site"
	"infra/cmd/skylab/internal/userinput"
	"infra/cmdsupport/cmdlib"
	"infra/libs/skylab/swarming"
)

const (
	dayInMinutes = 24 * 60

	// maxTasksPerModel is the maximum number of tasks that are allowed to be executing
	// at the same time for a given model.
	maxTasksPerModel = 1

	// maxTasksPerBoard is the maximum number of tasks that are allowed to be executing
	// at the same time for a given board. It is a completely independent cap from
	// maxTasksPerModel. A board lease does not count towards the model cap and vice versa.
	maxTasksPerBoard = 0

	boardLease    primaryLeaseDimensionType = "board"
	modelLease                              = "model"
	hostnameLease                           = "hostname"
)

// LeaseDut subcommand: Lease a DUT for debugging.
var LeaseDut = &subcommands.Command{
	UsageLine: "lease-dut HOST\n\tskylab lease-dut -model MODEL",
	ShortDesc: "lease DUT for debugging",
	LongDesc: `Lease DUT for debugging.

This subcommand's behavior is subject to change without notice.
Do not build automation around this subcommand.`,
	CommandRun: func() subcommands.CommandRun {
		c := &leaseDutRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		// Use a float so that large values passed on the command line are NOT wrapped.
		c.Flags.Float64Var(&c.leaseMinutes, "minutes", 60, "Duration of lease.")
		c.Flags.StringVar(&c.leaseReason, "reason", "", "The reason to perform this lease, it must match crbug.com/NNNN or b/NNNN.")
		// TODO(gregorynisbet):
		// If a model is provided, then we necessarily target DUT_POOL_QUOTA and only
		// repair-failed DUTs until a better policy can be implemented.
		c.Flags.StringVar(&c.model, "model", "", "Leases may optionally target a model instead of a hostname.")
		c.Flags.StringVar(&c.board, "board", "", "Leases may optionally target a board instead of a hostname.")
		// We allow arbitrary dimensions to be passed in via the -dim and/or -dims flags.
		// For maximum flexibility, both flags can take one or more key:val or key=val
		// pairs (separated by ","), and repeated/mixed flags are allowed. To keep the
		// docs intuitive, docstrings describe the more natural arg format for each flag.
		c.Flags.Var(flagx.Dims(&c.dims), "dim", "Single additional dimension in format key=value or key:value; may be specified multiple times.")
		c.Flags.Var(flagx.Dims(&c.dims), "dims", "List of additional dimensions in format key1=value1,key2=value2,... or key1:value1,key2:value2,... .")
		c.Flags.BoolVar(&c.evilLease, "evil-lease", false, "Evil lease allows the user to bypass the lease per model limit if there's an emergency.")
		return c
	},
}

type primaryLeaseDimensionType string

type leaseDutRun struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     skycmdlib.EnvFlags
	leaseMinutes float64
	leaseReason  string
	model        string
	board        string
	dims         map[string]string
	evilLease    bool
}

func (c *leaseDutRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *leaseDutRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	hasOneHostname := len(args) == 1
	hasModel := c.model != ""
	hasBoard := c.board != ""

	if !exactlyOne(hasOneHostname, hasModel, hasBoard) {
		return cmdlib.NewUsageError(c.Flags, "exactly one hostname or model or board required.")
	}
	if c.leaseMinutes < 0 {
		return cmdlib.NewUsageError(c.Flags, fmt.Sprintf("minutes to lease (%d) cannot be negative", int64(c.leaseMinutes)))
	}
	if c.leaseMinutes >= dayInMinutes {
		return cmdlib.NewUsageError(c.Flags, "Lease duration (%d minutes) cannot exceed 1 day [%d minutes]", int64(c.leaseMinutes), dayInMinutes)
	}
	if len(c.leaseReason) > 30 {
		return cmdlib.NewUsageError(c.Flags, "the lease reason is limited in 30 characters")
	}
	if userinput.ValidBug(c.leaseReason) {
		return cmdlib.NewUsageError(c.Flags, "the lease reason must match crbug.com/NNNN or b/NNNN")
	}

	ctx := cli.GetContext(a, c, env)

	leaseDuration := time.Duration(c.leaseMinutes) * time.Minute

	sc, err := c.newSwarmingClient(ctx)
	if err != nil {
		return err
	}
	bc, err := bb.NewClient(ctx, c.envFlags.Env().DUTLeaserBuilderInfo, c.authFlags)
	if err != nil {
		return err
	}

	var taskID int64
	switch {
	case hasOneHostname:
		oldhost := args[0]
		host := skycmdlib.FixSuspiciousHostname(oldhost)
		if host != oldhost {
			fmt.Fprintf(a.GetErr(), "correcting (%s) to (%s)\n", oldhost, host)
		}
		taskID, err = c.leaseDutByHostname(ctx, a, sc, bc, leaseDuration, host)
	case hasBoard:
		taskID, err = c.leaseDUTByBoard(ctx, a, sc, bc, leaseDuration)
	default:
		taskID, err = c.leaseDUTByModel(ctx, a, sc, bc, leaseDuration)
	}
	if err != nil {
		return err
	}

	dutName, err := c.waitForBuildStart(ctx, bc, taskID)
	if err != nil {
		return err
	}
	fqdn := dutNameToFQDN(dutName)
	fmt.Fprintf(a.GetOut(), "%s\n", fqdn)
	// TODO(ayatane): The time printed here may be off by the poll interval above.
	fmt.Fprintf(a.GetOut(), "DUT leased until %s\n", time.Now().Add(leaseDuration).Format(time.RFC1123))
	return nil
}

// leaseDutByHostname leases a DUT by hostname and schedules a follow-up repair task
func (c *leaseDutRun) leaseDutByHostname(ctx context.Context, a subcommands.Application, sc *swarming.Client, bc *bb.Client, leaseDuration time.Duration, host string) (taskID int64, err error) {
	ic, err := getUFSClient(ctx, &c.authFlags, c.envFlags.Env())
	if err != nil {
		return 0, err
	}

	// This is done to allow the lease task to be tagged with the model.
	// This allows to rate-limit the number of leases for a given model M by checking for leases with the label-model:M tag.
	model, err := getModelForHost(ctx, ic, host)
	if err != nil {
		return 0, err
	}
	fmt.Fprintf(a.GetErr(), "inferred model (%s)\n", model)
	var extraTags []string
	if model != "" {
		extraTags = append(extraTags, fmt.Sprintf("label-model:%s", model))
	}

	dims, tags, err := c.fullBBDimensionsAndTags(ctx, sc, hostnameLease, host, extraTags...)
	if err != nil {
		return 0, err
	}
	id, err := bc.ScheduleDUTLeaserBuild(ctx, dims, tags, int32(leaseDuration.Minutes()))
	if err != nil {
		return 0, err
	}
	fmt.Fprintf(a.GetOut(), "Created lease for host %s: %s\n", host, bc.BuildURL(id))
	fmt.Fprintf(a.GetOut(), "Waiting for task to start; lease isn't active yet\n")
	return id, nil
}

// leaseDutByModel leases a DUT by model. Any healthy DUT in the given model may be chosen by the task.
func (c *leaseDutRun) leaseDUTByModel(ctx context.Context, a subcommands.Application, sc *swarming.Client, bc *bb.Client, leaseDuration time.Duration) (taskID int64, err error) {
	tasks, err := sc.GetActiveLeaseTasksForModel(ctx, c.model)
	if err != nil {
		return 0, errors.Annotate(err, "computing existing leases").Err()
	}
	if maxTasksPerModel <= 0 {
		return 0, errors.Reason("Leases by model are disabled").Err()
	}
	if !c.evilLease && len(tasks) > maxTasksPerModel {
		return 0, fmt.Errorf("number of active tasks %d for model %q exceeds global limit for all users %d", len(tasks), c.model, maxTasksPerModel)
	}

	dims, tags, err := c.fullBBDimensionsAndTags(ctx, sc, modelLease, c.model)
	if err != nil {
		return 0, errors.Annotate(err, "generating dimensions and tags for board").Err()
	}

	id, err := bc.ScheduleDUTLeaserBuild(ctx, dims, tags, int32(leaseDuration.Minutes()))
	if err != nil {
		return 0, err
	}
	fmt.Fprintf(a.GetOut(), "Created lease for model %s: %s\n", c.model, bc.BuildURL(id))
	return id, nil
}

// leaseDUTbyBoard leases a DUT by board.
func (c *leaseDutRun) leaseDUTByBoard(ctx context.Context, a subcommands.Application, sc *swarming.Client, bc *bb.Client, leaseDuration time.Duration) (taskID int64, err error) {
	tasks, err := sc.GetActiveLeaseTasksForBoard(ctx, c.board)
	if err != nil {
		return 0, errors.Annotate(err, "computing existing lease for board").Err()
	}

	if maxTasksPerBoard <= 0 {
		return 0, errors.Reason("Leases by board are disabled").Err()
	}
	if len(tasks) > maxTasksPerBoard {
		return 0, errors.Reason("number of active tasks %d for board %q exceeds global limit for all users %d", len(tasks), c.board, maxTasksPerBoard).Err()
	}

	dims, tags, err := c.fullBBDimensionsAndTags(ctx, sc, boardLease, c.board)
	if err != nil {
		return 0, errors.Annotate(err, "generating dimensions and tags for board").Err()
	}

	id, err := bc.ScheduleDUTLeaserBuild(ctx, dims, tags, int32(leaseDuration.Minutes()))
	if err != nil {
		return 0, err
	}
	fmt.Fprintf(a.GetOut(), "Created lease for board %s: %s\n", c.board, bc.BuildURL(id))
	return id, nil
}

// bbDimensionsAndTags creates the full Buildbucket dimensions and tags used to lease a DUT of
// the specified board/model/hostname dimension.
func (c *leaseDutRun) fullBBDimensionsAndTags(ctx context.Context, sc *swarming.Client, primaryDimType primaryLeaseDimensionType, primaryDim string, extraTags ...string) (map[string]string, []string, error) {
	fullDims := make(map[string]string)
	tags := append(
		extraTags,
		"qs_account:leases",
		fmt.Sprintf("lease-by:%s", primaryDimType),
		fmt.Sprintf("lease-reason:%s", c.leaseReason),
		"skylab-tool:lease",
	)

	if primaryDimType == hostnameLease {
		botID, err := sc.DutNameToBotID(ctx, primaryDim)
		if err != nil {
			return nil, nil, err
		}
		fullDims["id"] = botID
		tags = append(tags, fmt.Sprintf("dut-name:%s", primaryDim))
	} else {
		fullDims[fmt.Sprintf("label-%s", primaryDimType)] = primaryDim
		fullDims["label-pool"] = "DUT_POOL_QUOTA"
		fullDims["dut_state"] = "ready"
		tags = append(tags, fmt.Sprintf("%s:%s", primaryDimType, primaryDim))
	}

	// Any arbitrary dimensions passed from the command line should be included in both
	// BB dimensions and tags.
	fullDims = mergeDims(&fullDims, &c.dims)
	tags = append(tags, dimsToTags(&c.dims)...)

	return fullDims, tags, nil
}

// newSwarmingClient creates a new swarming client.
func (c *leaseDutRun) newSwarmingClient(ctx context.Context) (*swarming.Client, error) {
	h, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return nil, err
	}
	e := c.envFlags.Env()
	client, err := swarming.NewClient(h, e.SwarmingService)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// waitForBuildStart waits for the dut_leaser build with the given id to start and
// returns the unqualified hostname of the DUT running the task.
func (c *leaseDutRun) waitForBuildStart(ctx context.Context, client *bb.Client, id int64) (string, error) {
	for {
		build, err := client.GetBuild(ctx, id)
		if err != nil {
			return "", err
		}
		switch s := build.Status; s {
		case buildbucket_pb.Status_SCHEDULED:
			time.Sleep(10 * time.Second)
		case buildbucket_pb.Status_STARTED:
			dutName := build.DUTName
			if dutName == "" {
				return "", errors.Reason("No dut_name for build %d", id).Err()
			}
			return dutName, nil
		default:
			return "", errors.Reason("Got unexpected build status %s", buildbucket_pb.Status_name[int32(s)]).Err()
		}
	}
}

// exactlyOne counts the number of true booleans and returns whether it is exactly one
func exactlyOne(bools ...bool) bool {
	count := 0
	for _, b := range bools {
		if b {
			count++
		}
		if count > 1 {
			return false
		}
	}
	return count == 1
}

// dutNameToFQDN converts a dutName of the form "HOSTNAME" to "HOSTNAME.cros.corp.google.com".
// dutNameToFQDN assumes that a dutName is a valid DUT name, if it is not, then the output is arbitrary.
func dutNameToFQDN(dutName string) string {
	return fmt.Sprintf("%s.cros.corp.google.com", dutName)
}

// convertTags takes a map of dimensions and converts it to a slice of strings in
// swarming dimension format.
func dimsToTags(dims *map[string]string) []string {
	tags := make([]string, len(*dims))
	i := 0
	for k, v := range *dims {
		tags[i] = fmt.Sprintf("%s:%s", k, v)
		i++
	}
	return tags
}

// mergeDims merges two sets of dimensions, taking values from the first set if there is key overlap.
func mergeDims(first, second *map[string]string) map[string]string {
	merged := make(map[string]string)
	for k, v := range *first {
		merged[k] = v
	}
	for k, v := range *second {
		if _, keyExists := merged[k]; !keyExists {
			merged[k] = v
		}
	}
	return merged
}
