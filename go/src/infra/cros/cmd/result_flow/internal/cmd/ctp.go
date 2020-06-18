// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/logging"

	"infra/cmdsupport/cmdlib"
	"infra/cros/cmd/result_flow/internal/message"
	"infra/cros/cmd/result_flow/internal/site"
)

// CTP subcommand pipelines CTP builds to analytics BQ table represented in the form
// of test_platform/analytics/TestPlanRun.
var CTP = &subcommands.Command{
	UsageLine: `ctp [FLAGS...]`,
	ShortDesc: "Upload CTP Build data to Bigquery",
	LongDesc: `ctp command catches a set of CTP builds, and
	uploads the build data to Bigquery in the format of
	test_platform/analytics/TestPlanRun proto.`,
	CommandRun: func() subcommands.CommandRun {
		c := &ctpFlowRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)

		c.Flags.StringVar(&c.inputPath, "input_json", "", "Path that contains JSON encoded test_platform.result_fow.CTPRequest")
		c.Flags.StringVar(&c.outputPath, "output_json", "", "Path to write JSON encoded test_platform.result_flow.CTPResponse to")
		return c
	},
}

type ctpFlowRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	inputPath  string
	outputPath string

	deadline time.Time
	source   *result_flow.Source
	target   *result_flow.Target
}

func (c *ctpFlowRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *ctpFlowRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.loadCTPRequest(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)

	var cf context.CancelFunc
	logging.Infof(ctx, "Running with deadline %s (current time: %s)", c.deadline.UTC(), time.Now().UTC())
	ctx, cf = context.WithDeadline(ctx, c.deadline)
	defer cf()

	s, err := runWithDeadline(
		ctx,
		func(ch chan state) {
			c.pipelineRun(ctx, ch)
		},
	)
	werr := writeJSONPb(c.outputPath, &result_flow.CTPResponse{State: s})
	if err != nil {
		return err
	}
	return werr
}

func (c *ctpFlowRun) pipelineRun(ctx context.Context, ch chan state) {
	defer close(ch)
	mClient, err := message.NewClient(ctx, c.source.GetPubsub())
	if err != nil {
		ch <- state{result_flow.State_FAILED, err}
		return
	}
	defer mClient.Close()
	msgs, err := mClient.PullMessages(ctx)
	if err != nil {
		ch <- state{result_flow.State_FAILED, err}
		return
	}
	bIDs := message.ToBuildIDs(ctx, msgs)
	// TODO(linxinan): Next CL will call BB to get those builds.
	// At this stage, the subcommand print out the build ID pulled
	// from Pub/Sub.
	for _, bID := range bIDs {
		logging.Infof(ctx, "Fetched Build IDs from PubSub: %d", bID)
	}

	if err = mClient.AckMessages(ctx, msgs); err != nil {
		ch <- state{result_flow.State_FAILED, err}
		return
	}
	ch <- state{result_flow.State_SUCCEEDED, nil}
}

func (c *ctpFlowRun) loadCTPRequest() error {
	var (
		r   result_flow.CTPRequest
		err error
	)
	if err = readJSONPb(c.inputPath, &r); err != nil {
		return err
	}
	if c.source, err = verifySource(r.GetCtp()); err != nil {
		return err
	}
	if c.target, err = verifyTarget(r.GetTestPlanRun()); err != nil {
		return err
	}
	c.deadline = getDeadline(r.GetDeadline(), site.DefaultDeadlineSeconds)
	return nil
}
