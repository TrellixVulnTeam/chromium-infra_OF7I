// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"strings"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"

	"infra/cmd/cros_test_platform/internal/site"
	"infra/cmd/cros_test_platform/internal/trafficsplit"

	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/migration/scheduler"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/api/gitiles"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
)

// SchedulerTrafficSplit implements the `scheduler-traffic-split` subcommand.
var SchedulerTrafficSplit = &subcommands.Command{
	UsageLine: "scheduler-traffic-split -input_json /path/to/input.json -output_json /path/to/output.json",
	ShortDesc: "Determine traffic split between backend schedulers.",
	LongDesc: `Determine traffic split between backend schedulers, i.e. Autotest vs Skylab.

Step input and output is JSON encoded protobuf defined at
https://chromium.googlesource.com/chromiumos/infra/proto/+/master/src/test_platform/steps/scheduler_traffic_split.proto`,
	CommandRun: func() subcommands.CommandRun {
		c := &schedulerTrafficSplitRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.Flags.StringVar(&c.inputPath, "input_json", "", "Path that contains JSON encoded test_platform.steps.SchedulerTrafficSplitRequests")
		c.Flags.StringVar(&c.outputPath, "output_json", "", "Path where JSON encoded test_platform.steps.SchedulerTrafficSplitResponses should be written.")
		c.Flags.BoolVar(&c.directAllToSkylab, "rip-cautotest", false, "Cautotest is now at peace. Use a simple forwarding rule to send all traffic to Skylab.")
		c.Flags.BoolVar(&c.tagged, "tagged", true, "Transitional flag to enable tagged requests and responses.")
		return c
	},
}

type schedulerTrafficSplitRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	inputPath  string
	outputPath string

	// A fast-path flag that replaces the traffic splitter logic with a mostly
	// trivial redirection to Skylab.
	directAllToSkylab bool

	// TODO(crbug.com/1002941) Completely transition to tagged requests only, once
	// - recipe has transitioned to using tagged requests
	// - autotest-execute has been deleted (this just reduces the work required).
	tagged      bool
	orderedTags []string
}

func (c *schedulerTrafficSplitRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	err := c.innerRun(a, args, env)
	if err != nil {
		fmt.Fprintf(a.GetErr(), "%s\n", err)
	}
	return exitCode(err)
}

func (c *schedulerTrafficSplitRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.processCLIArgs(args); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	ctx = setupLogging(ctx)

	requests, err := c.readRequests()
	if err != nil {
		return err
	}
	if len(requests) == 0 {
		return errors.Reason("zero requests").Err()
	}

	if c.directAllToSkylab {
		return c.sendAllToSkylab(requests)
	}
	return c.respondUsingConfig(ctx, requests)
}

func (c *schedulerTrafficSplitRun) respondUsingConfig(ctx context.Context, requests []*steps.SchedulerTrafficSplitRequest) error {
	if err := ensureIdenticalConfigs(requests); err != nil {
		return err
	}

	split, err := c.getTrafficSplitConfig(ctx, requests[0].Config)
	if err != nil {
		return err
	}

	resps := make([]*steps.SchedulerTrafficSplitResponse, len(requests))
	merr := errors.NewMultiError()
	for i, r := range requests {
		if resp, err := trafficsplit.ApplyToRequest(r.Request, split); err == nil {
			resps[i] = resp
		} else {
			logPotentiallyRelevantRules(ctx, r.Request, split.Rules)
			merr = append(merr, err)
		}
	}
	if merr.First() != nil {
		return merr
	}

	if autotestResponseCount(resps) > 0 && len(resps) > 1 {
		return errors.Reason("multiple requests with autotest backend: %s", resps).Err()
	}
	return c.writeResponses(resps)
}

func (c *schedulerTrafficSplitRun) sendAllToSkylab(requests []*steps.SchedulerTrafficSplitRequest) error {
	resps := make([]*steps.SchedulerTrafficSplitResponse, len(requests))
	for i, r := range requests {
		resps[i] = c.sendToSkylab(r.Request)
	}
	return c.writeResponses(resps)
}

func (c *schedulerTrafficSplitRun) sendToSkylab(req *test_platform.Request) *steps.SchedulerTrafficSplitResponse {
	var dst test_platform.Request
	proto.Merge(&dst, req)
	setQuotaAccountForLegacyPools(&dst)
	return &steps.SchedulerTrafficSplitResponse{
		SkylabRequest: &dst,
	}
}

// TODO(crbug.com/1026367) Once CTP stops receiving requests targeted at these
// legacy pools, drop the traffic splitter step entirely.
// The main source of these legacy requests are release builders on old
// branches.
var quotaAccountsForLegacyPools = map[test_platform.Request_Params_Scheduling_ManagedPool]string{
	test_platform.Request_Params_Scheduling_MANAGED_POOL_BVT:        "legacypool-bvt",
	test_platform.Request_Params_Scheduling_MANAGED_POOL_CONTINUOUS: "pfq",
	test_platform.Request_Params_Scheduling_MANAGED_POOL_CQ:         "cq",
	test_platform.Request_Params_Scheduling_MANAGED_POOL_SUITES:     "legacypool-suites",
}

func setQuotaAccountForLegacyPools(req *test_platform.Request) {
	if qa, ok := quotaAccountsForLegacyPools[req.GetParams().GetScheduling().GetManagedPool()]; ok {
		req.Params.Scheduling.Pool = &test_platform.Request_Params_Scheduling_QuotaAccount{
			QuotaAccount: qa,
		}
	}
}

func (c *schedulerTrafficSplitRun) processCLIArgs(args []string) error {
	if len(args) > 0 {
		return errors.Reason("have %d positional args, want 0", len(args)).Err()
	}
	if c.inputPath == "" {
		return errors.Reason("-input_json not specified").Err()
	}
	if c.outputPath == "" {
		return errors.Reason("-output_json not specified").Err()
	}
	return nil
}

func (c *schedulerTrafficSplitRun) readRequests() ([]*steps.SchedulerTrafficSplitRequest, error) {
	var rs steps.SchedulerTrafficSplitRequests
	if err := readRequest(c.inputPath, &rs); err != nil {
		return nil, err
	}
	if !c.tagged {
		return rs.Requests, nil
	}
	ts, reqs := c.unzipTaggedRequests(rs.TaggedRequests)
	c.orderedTags = ts
	return reqs, nil
}

func (c *schedulerTrafficSplitRun) unzipTaggedRequests(trs map[string]*steps.SchedulerTrafficSplitRequest) ([]string, []*steps.SchedulerTrafficSplitRequest) {
	var ts []string
	var rs []*steps.SchedulerTrafficSplitRequest
	for t, r := range trs {
		ts = append(ts, t)
		rs = append(rs, r)
	}
	return ts, rs
}

func ensureIdenticalConfigs(rs []*steps.SchedulerTrafficSplitRequest) error {
	if len(rs) == 0 {
		return nil
	}
	c := rs[0].GetConfig()
	for _, o := range rs[1:] {
		if !proto.Equal(c, o.GetConfig()) {
			return errors.Reason("mismatched configs: %s vs %s", c, o.GetConfig()).Err()
		}
	}
	return nil
}

func autotestResponseCount(rs []*steps.SchedulerTrafficSplitResponse) int {
	c := 0
	for _, r := range rs {
		if r.GetAutotestRequest() != nil {
			c++
		}
	}
	return c
}

func (c *schedulerTrafficSplitRun) writeResponses(resps []*steps.SchedulerTrafficSplitResponse) error {
	r := &steps.SchedulerTrafficSplitResponses{
		Responses: resps,
	}
	if c.tagged {
		r.TaggedResponses = c.zipTaggedResponses(c.orderedTags, resps)
	}
	return writeResponse(c.outputPath, r)
}

func (c *schedulerTrafficSplitRun) zipTaggedResponses(ts []string, rs []*steps.SchedulerTrafficSplitResponse) map[string]*steps.SchedulerTrafficSplitResponse {
	if len(ts) != len(rs) {
		panic(fmt.Sprintf("got %d responses for %d tags (%s)", len(rs), len(ts), ts))
	}
	m := make(map[string]*steps.SchedulerTrafficSplitResponse)
	for i := range ts {
		m[ts[i]] = rs[i]
	}
	return m
}

func (c *schedulerTrafficSplitRun) getTrafficSplitConfig(ctx context.Context, config *config.Config_SchedulerMigration) (*scheduler.TrafficSplit, error) {
	g, err := c.newGitilesClient(ctx, config.GitilesHost)
	if err != nil {
		return nil, errors.Annotate(err, "get traffic split config").Err()
	}
	text, err := c.downloadTrafficSplitConfig(ctx, g, config)
	if err != nil {
		return nil, errors.Annotate(err, "get traffic split config").Err()
	}
	var split scheduler.TrafficSplit
	if err := unmarshaller.Unmarshal(strings.NewReader(text), &split); err != nil {
		return nil, errors.Annotate(err, "get traffic split config").Err()
	}
	return &split, nil
}

func (c *schedulerTrafficSplitRun) newGitilesClient(ctx context.Context, host string) (gitilespb.GitilesClient, error) {
	h, err := newAuthenticatedHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return nil, errors.Annotate(err, "new gitiles client").Err()
	}
	return gitiles.NewRESTClient(h, host, true)
}

// downloadTrafficSplitConfig returns the contents of the config downloaded from Gitiles.
func (c *schedulerTrafficSplitRun) downloadTrafficSplitConfig(ctx context.Context, client gitilespb.GitilesClient, config *config.Config_SchedulerMigration) (string, error) {
	res, err := client.DownloadFile(ctx, &gitilespb.DownloadFileRequest{
		Project:    config.GitProject,
		Committish: config.Commitish,
		Path:       config.FilePath,
		Format:     gitilespb.DownloadFileRequest_TEXT,
	})
	if err != nil {
		return "", errors.Annotate(err, "download from gitiles").Err()
	}
	return res.Contents, nil
}

func logPotentiallyRelevantRules(ctx context.Context, request *test_platform.Request, rules []*scheduler.Rule) {
	f := trafficsplit.NewRuleFilter(rules)
	logger := logging.Get(ctx)
	logger.Warningf("No matching rule found for %s. Printing partially matching rules...", request)

	m := request.GetParams().GetHardwareAttributes().GetModel()
	if pr := f.ForModel(m); len(pr) > 0 {
		logger.Infof("Following rules match requested model: %s", formatFirstFewRules(pr))
	} else {
		logger.Warningf("No rules matched requested model %s.", m)
	}

	b := request.GetParams().GetSoftwareAttributes().GetBuildTarget().GetName()
	if pr := f.ForBuildTarget(b); len(pr) > 0 {
		logger.Infof("Following rules match requested buildTarget: %s", formatFirstFewRules(pr))
	} else {
		logger.Warningf("No rules matched requested build target %s.", b)
	}

	s := request.GetParams().GetScheduling()
	if pr := f.ForScheduling(s); len(pr) > 0 {
		logger.Infof("Following rules match requested scheduling: %s", formatFirstFewRules(pr))
	} else {
		logger.Warningf("No rules matched requested scheduling %s.", s)
	}
}

func formatFirstFewRules(rules []*scheduler.Rule) string {
	const numRulesToPrint = 5
	rulesToPrint := rules
	if len(rulesToPrint) > numRulesToPrint {
		rulesToPrint = rulesToPrint[:numRulesToPrint]
	}
	s := fmt.Sprintf("%v", rulesToPrint)
	if len(s) > numRulesToPrint {
		s = fmt.Sprintf("%s... [%d more]", s, len(s)-numRulesToPrint)
	}
	return s
}
