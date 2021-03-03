// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"time"

	"infra/cros/cmd/result_flow/internal/bb"
	"infra/cros/cmd/result_flow/internal/bq"
	"infra/cros/cmd/result_flow/internal/message"
	"infra/cros/cmd/result_flow/internal/site"
	"infra/cros/cmd/result_flow/internal/transform"

	"cloud.google.com/go/bigquery"
	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"
	"go.chromium.org/luci/auth/client/authcli"
	lucibq "go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/logging"
	pubsubpb "google.golang.org/genproto/googleapis/pubsub/v1"
)

// PipeCTPData subcommand pipelines CTP builds to analytics BQ table represented in the form
// of test_platform/analytics/TestPlanRun.
var PipeCTPData = &subcommands.Command{
	UsageLine: `pipe-ctp-data [FLAGS...]`,
	ShortDesc: "Pipe CTP Build data to Bigquery",
	LongDesc: `pipe-ctp-data command catches a set of CTP builds, and
	pipes the build data to Bigquery in the format of
	test_platform/analytics/TestPlanRun proto.`,
	CommandRun: func() subcommands.CommandRun {
		c := &pipeCTPDataRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)

		c.Flags.StringVar(&c.inputPath, "input_json", "", "Path that contains JSON encoded test_platform.result_fow.CTPRequest")
		c.Flags.StringVar(&c.outputPath, "output_json", "", "Path to write JSON encoded test_platform.result_flow.CTPResponse to")
		return c
	},
}

type pipeCTPDataRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	inputPath  string
	outputPath string

	deadline time.Time
	source   *result_flow.Source
	target   *result_flow.Target

	mClient  message.Client
	bbClient bb.Client
	bqClient bq.Inserter
}

func (c *pipeCTPDataRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), err.Error())
		return 1
	}
	return 0
}

func (c *pipeCTPDataRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.loadCTPRequest(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)

	authOpts, err := c.authFlags.Options()
	if err != nil {
		return err
	}
	clientOpts, err := newGRPCClientOptions(ctx, authOpts)
	if err != nil {
		return err
	}
	httpClient, err := newHTTPClient(ctx, authOpts)
	if err != nil {
		return err
	}

	// Pubsub client
	c.mClient, err = message.NewClient(ctx, c.source.GetPubsub(), site.CTPBatchSize, clientOpts)
	if err != nil {
		return err
	}
	defer c.mClient.Close()

	// Buildbucket client
	c.bbClient, err = bb.NewClient(
		ctx,
		c.source.GetBb(),
		c.source.GetFields(),
		httpClient,
	)
	if err != nil {
		return err
	}

	// Bigquery inserter for TestPlanRun.
	c.bqClient, err = bq.NewInserter(ctx,
		bq.Options{
			Target:     c.target.GetBq(),
			HTTPClient: httpClient,
		},
	)
	if err != nil {
		return err
	}
	defer c.bqClient.Close()

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

func (c *pipeCTPDataRun) pipelineRun(ctx context.Context, ch chan state) {
	defer close(ch)

	builds, msgsByBuildID, err := fetchBuilds(
		ctx,
		int(c.source.GetPubsub().GetMaxReceivingMessages()),
		c.mClient,
		c.bbClient,
	)
	if err != nil {
		ch <- state{result_flow.State_FAILED, err}
		return
	}
	if len(builds) == 0 {
		ch <- state{result_flow.State_SUCCEEDED, nil}
		return
	}
	logging.Infof(ctx, "fetched %d builds from Buildbucket", len(builds))

	var processed []*pubsubpb.ReceivedMessage
	for _, build := range builds {
		if shouldSkipMessage(msgsByBuildID[build.Id], build) {
			logging.Infof(ctx, "skip build %d: the build finished running tests but is not marked complete yet", build.Id)
			continue
		}
		cBuild, err := transform.LoadCTPBuildBucketResp(ctx, build, c.source.GetBb())
		if err != nil {
			logging.Errorf(ctx, "failed to extract data from build: %v", err)
			continue
		}
		if err = c.bqClient.Insert(ctx, toRows(ctx, cBuild)...); err != nil {
			logging.Errorf(ctx, "failed to upload build data to Bigquery: %v", err)
		}
		processed = append(processed, msgsByBuildID[build.Id])
	}
	c.bqClient.CloseAndDrain(ctx)

	logging.Infof(ctx, "processed %d builds of %d total fetched", len(processed), len(builds))
	if err = c.mClient.AckMessages(ctx, processed); err != nil {
		ch <- state{result_flow.State_FAILED, err}
		return
	}
	ch <- state{result_flow.State_SUCCEEDED, nil}
}

func (c *pipeCTPDataRun) loadCTPRequest() error {
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

func toRows(ctx context.Context, b transform.CTPBuildResults) []bigquery.ValueSaver {
	var rows []bigquery.ValueSaver
	for _, t := range b.ToTestPlanRuns(ctx) {
		// Reflect status in the InsertID, because for some build its pre-execution and
		// post-execution messages may get processed at same time.
		rows = append(rows, &lucibq.Row{Message: t, InsertID: fmt.Sprintf("%s/%s", t.GetUid(), t.GetStatus().GetValue())})
	}
	return rows
}
