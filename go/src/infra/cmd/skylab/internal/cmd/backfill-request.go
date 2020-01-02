// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/common/logging"

	"infra/cmd/skylab/internal/bb"
	"infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/site"
	"infra/cmd/skylab/internal/userinput"
)

// BackfillRequest subcommand: Backfill unsuccessful results for a previous
// request.
var BackfillRequest = &subcommands.Command{
	UsageLine: `backfill-request [FLAGS...]`,
	ShortDesc: "backfill unsuccessful results for a previous request",
	LongDesc: `Backfill unsuccessful results for a previous request.

This command creates a new cros_test_platform request to backfill results from
unsuccessful (expired, timed out, or failed) tasks from a previous build.

The backfill request uses the same parameters as the original request (model,
pool, build etc.). The backfill request attempts to minimize unnecessary task
execution by skipping tasks that have succeeded previously when possible.

This command does not wait for the build to start running.`,
	CommandRun: func() subcommands.CommandRun {
		c := &backfillRequestRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.Int64Var(&c.buildID, "id", -1, "Search for original build with this ID. Mutually exclusive with -tag.")
		c.Flags.Var(flag.StringSlice(&c.buildTags), "tag", "Search for original build matching given tag. May be used multiple times to provide more tags to match. Mutually exclusive with -id")
		c.Flags.BoolVar(&c.highestPriority, "highest-priority", false, "Create backfill tasks at highest priority. This will displace legitimate prod tasks. Use with care.")
		return c
	},
}

type backfillRequestRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  cmdlib.EnvFlags

	buildID         int64
	buildTags       []string
	highestPriority bool

	bbClient *bb.Client
}

func (c *backfillRequestRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a.GetErr(), err)
		return 1
	}
	return 0
}

func (c *backfillRequestRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	if err := c.setNewBBClient(ctx); err != nil {
		return err
	}

	originalBuilds, err := c.getOriginalBuilds(ctx)
	if err != nil {
		return err
	}

	switch {
	case len(originalBuilds) == 0:
		return errors.Reason("no matching build found").Err()
	case len(originalBuilds) > 1:
		if !c.confirmMultileBuildsOK(a, originalBuilds) {
			return nil
		}
	default:
	}

	var merr errors.MultiError
	for _, b := range originalBuilds {
		target := b

		backfills, err := c.getSortedBackfillBuildsFor(ctx, b)
		if err != nil {
			logging.Errorf(ctx, "Failed to find existing backfill requests for %s: %s", b, err)
			merr = append(merr, err)
			continue
		}

		if len(backfills) > 0 {
			if el := backfills[0]; isInFlight(el) {
				logging.Infof(ctx, "Build %s already in flight to backfill %s", c.bbClient.BuildURL(el.ID), c.bbClient.BuildURL(b.ID))
				continue
			}
			if t := c.getFirstWithValidBackfillRequest(backfills); t != nil {
				target = t
				logging.Debugf(ctx, "Backfilling a previous backfill attempt %s for %s", c.bbClient.BuildURL(target.ID), c.bbClient.BuildURL(b.ID))
			}
		}

		id, err := c.scheduleBackfillBuild(ctx, target)
		if err != nil {
			logging.Errorf(ctx, "Failed to create backfill request for %s: %s", b, err)
			merr = append(merr, err)
			continue
		}
		logging.Infof(ctx, "Scheduled %s to backfill %s", c.bbClient.BuildURL(id), c.bbClient.BuildURL(b.ID))

	}
	return merr.First()
}

func isInFlight(b *bb.Build) bool {
	return b.Status == buildbucketpb.Status_SCHEDULED || b.Status == buildbucketpb.Status_STARTED
}

// validateArgs ensures that the command line arguments are
func (c *backfillRequestRun) validateArgs() error {
	if c.Flags.NArg() != 0 {
		return cmdlib.NewUsageError(c.Flags, fmt.Sprintf("got %d positional arguments, want 0", c.Flags.NArg()))
	}
	switch {
	case c.isBuildIDSet() && c.isBuildTagsSet():
		return cmdlib.NewUsageError(c.Flags, "use only one of -id and -tag")
	case !(c.isBuildIDSet() || c.isBuildTagsSet()):
		return cmdlib.NewUsageError(c.Flags, "must use one of -id or -tag")
	}
	return nil
}

func (c *backfillRequestRun) isBuildIDSet() bool {
	// The default value of -1 is nonsensical.
	return c.buildID > 0
}

func (c *backfillRequestRun) isBuildTagsSet() bool {
	return len(c.buildTags) > 0
}

func (c *backfillRequestRun) setNewBBClient(ctx context.Context) error {
	client, err := bb.NewClient(ctx, c.envFlags.Env(), c.authFlags)
	if err == nil {
		c.bbClient = client
	}
	return err
}

func (c *backfillRequestRun) getOriginalBuilds(ctx context.Context) ([]*bb.Build, error) {
	if c.isBuildIDSet() {
		b, err := c.getOriginalBuildByID(ctx)
		return []*bb.Build{b}, err
	}
	return c.getOriginalBuildsByTags(ctx)
}

func (c *backfillRequestRun) getOriginalBuildByID(ctx context.Context) (*bb.Build, error) {
	b, err := c.bbClient.GetBuild(ctx, c.buildID)
	if err != nil {
		return nil, err
	}
	if isBackfillBuild(b) {
		return nil, errors.Reason("build ID %d is a backfill build", c.buildID).Err()
	}
	return b, nil
}

func isBackfillBuild(b *bb.Build) bool {
	for _, t := range b.Tags {
		if strings.HasPrefix(t, "backfill:") {
			return true
		}
	}
	return false
}

const bbBuildSearchLimit = 100

func (c *backfillRequestRun) getOriginalBuildsByTags(ctx context.Context) ([]*bb.Build, error) {
	builds, err := c.bbClient.SearchBuildsByTags(ctx, bbBuildSearchLimit, c.buildTags...)
	if err != nil {
		return nil, err
	}
	return filterOriginalBuilds(builds), nil
}

func filterOriginalBuilds(builds []*bb.Build) []*bb.Build {
	filtered := make([]*bb.Build, 0, len(builds))
	for _, b := range builds {
		if isOriginalBuild(b) {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

func isOriginalBuild(b *bb.Build) bool {
	return !isBackfillBuild(b)
}

// getSortedBackfillBuildsFor returns a list of backfill builds for the given
// build, sorted reverse-chronologically by creation time (latest first).
//
// getSortedBackfillBuildsFor returns nil (and no error) if no backfill builds
// are found.
func (c *backfillRequestRun) getSortedBackfillBuildsFor(ctx context.Context, b *bb.Build) ([]*bb.Build, error) {
	builds, err := c.bbClient.SearchBuildsByTags(ctx, bbBuildSearchLimit, backfillTags(b.Tags, b.ID)...)
	if err != nil {
		return nil, errors.Annotate(err, "get sorted backfill builds for %d", b.ID).Err()
	}
	if len(builds) == 0 {
		return nil, nil
	}
	// buildbucket builds IDs are monotonically decreasing.
	// The build with the smallest ID is the latest.
	sort.Slice(builds, func(i, j int) bool {
		return builds[i].ID < builds[j].ID
	})
	return builds, nil
}

// getFirstWithValidBackfillRequest returns the first build in the slice with
// a valid backfill request.
//
// getFirstWithValidBackfillRequest returns nil if no such build is found.
func (c *backfillRequestRun) getFirstWithValidBackfillRequest(bs []*bb.Build) *bb.Build {
	for _, b := range bs {
		if b.BackfillRequests != nil {
			return b
		}
	}
	return nil
}

func (c *backfillRequestRun) confirmMultileBuildsOK(a subcommands.Application, builds []*bb.Build) bool {
	prompt := userinput.CLIPrompt(a.GetOut(), os.Stdin, false)
	return prompt(fmt.Sprintf("Found %d builds to backfill. Create requests for them all [y/N]? ", len(builds)))
}

func (c *backfillRequestRun) scheduleBackfillBuild(ctx context.Context, original *bb.Build) (int64, error) {
	var reqs map[string]*test_platform.Request
	switch {
	case original.BackfillRequests != nil:
		reqs = original.BackfillRequests
	case original.Requests != nil:
		logging.Debugf(ctx, "Original build %d has no backfill requests. Using original requests instead.", original.ID)
		reqs = original.Requests
	case original.Request != nil:
		logging.Debugf(ctx, "Original build %d has no backfill requests. Using original request instead.", original.ID)
		reqs = map[string]*test_platform.Request{"default": original.Request}
	default:
		return -1, errors.Reason("schedule backfill: build %d has no request to clone", original.ID).Err()
	}

	if c.highestPriority {
		bumpPriority(reqs)
	}

	ID, err := c.bbClient.ScheduleBuild(ctx, reqs, backfillTags(original.Tags, original.ID))
	if err != nil {
		return -1, errors.Annotate(err, "schedule backfill").Err()
	}
	return ID, nil
}

const highestTestTaskPriority = 50

func bumpPriority(reqs map[string]*test_platform.Request) {
	for _, req := range reqs {
		sc := req.GetParams().GetScheduling()
		if sc != nil {
			sc.Priority = highestTestTaskPriority
		}
	}
}

func backfillTags(tags []string, originalID int64) []string {
	ntags := make([]string, 0, len(tags))
	for _, t := range tags {
		if isSkylabToolTag(t) {
			continue
		}
		ntags = append(ntags, t)
	}
	return append(ntags, "skylab-tool:backfill-request", fmt.Sprintf("backfill:%d", originalID))
}

func isSkylabToolTag(t string) bool {
	return strings.HasPrefix(t, "skylab-tool:")
}
