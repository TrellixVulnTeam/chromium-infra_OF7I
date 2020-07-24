// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"

	"time"

	"cloud.google.com/go/bigquery"

	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"
	"go.chromium.org/luci/auth/client/authcli"
	lucibq "go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/logging"

	"infra/cros/cmd/result_flow/internal/bb"
	"infra/cros/cmd/result_flow/internal/bq"
	"infra/cros/cmd/result_flow/internal/message"
	"infra/cros/cmd/result_flow/internal/site"
	"infra/cros/cmd/result_flow/internal/transform"
)

// PipeTestRunnerData subcommand pipes test runner build to test_platform/analytics/TestRun
// and test_platform/analytics/TestCaseResult for analytics usage.
var PipeTestRunnerData = &subcommands.Command{
	UsageLine: `pipe-test-runner-data [FLAGS...]`,
	ShortDesc: "Pipe Test Runner Build data to Bigquery",
	LongDesc: `pipe-test-runner-data command catches test runner builds, and
	and pipes their data to test_platform/analytics/TestRun as well as
	test_platform/analytics/TestCaseResult for analytics usage.`,
	CommandRun: func() subcommands.CommandRun {
		c := &pipeTestRunnerDataRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)

		c.Flags.StringVar(&c.inputPath, "input_json", "", "Path that contains JSON encoded test_platform.result_fow.TestRunnerRequest")
		c.Flags.StringVar(&c.outputPath, "output_json", "", "Path to write JSON encoded test_platform.result_flow.TestRunnerResponse to")
		return c
	},
}

type pipeTestRunnerDataRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	inputPath  string
	outputPath string

	deadline       time.Time
	source         *result_flow.Source
	testRun        *result_flow.Target
	testCaseResult *result_flow.Target

	mClient          message.Client
	bbClient         bb.Client
	bqTestRunClient  bq.Inserter
	bqTestCaseClient bq.Inserter
}

func (c *pipeTestRunnerDataRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), err.Error())
		return 1
	}
	return 0
}

func (c *pipeTestRunnerDataRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.loadTestRunnerRequest(); err != nil {
		return err
	}
	ctx, cf := context.WithDeadline(cli.GetContext(a, c, env), c.deadline)
	logging.Infof(ctx, "Running with deadline %s (current time: %s)", c.deadline.UTC(), time.Now().UTC())
	defer cf()

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
	c.mClient, err = message.NewClient(ctx, c.source.GetPubsub(), clientOpts)
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

	// Bigquery inserter for TestRun.
	c.bqTestRunClient, err = bq.NewInserter(ctx,
		bq.Options{
			Target:     c.testRun.GetBq(),
			HTTPClient: httpClient,
		},
	)
	if err != nil {
		return err
	}
	defer c.bqTestRunClient.Close()

	// Bigquery inserter for TestCaseResult.
	c.bqTestCaseClient, err = bq.NewInserter(ctx,
		bq.Options{
			Target:     c.testCaseResult.GetBq(),
			HTTPClient: httpClient,
		},
	)
	if err != nil {
		return err
	}
	defer c.bqTestCaseClient.Close()

	s, err := runWithDeadline(
		ctx,
		func(ch chan state) {
			c.pipelineRun(ctx, ch)
		},
	)
	werr := writeJSONPb(c.outputPath, &result_flow.TestRunnerResponse{State: s})
	if err != nil {
		return err
	}
	return werr
}

func (c *pipeTestRunnerDataRun) pipelineRun(ctx context.Context, ch chan state) {
	defer close(ch)

	msgs, err := c.mClient.PullMessages(ctx)
	if err != nil {
		ch <- state{result_flow.State_FAILED, err}
		return
	}
	buildIDMap := message.ExtractBuildIDMap(ctx, msgs)

	builds, err := c.bbClient.GetTargetBuilds(ctx, getBuildIDs(buildIDMap))
	if err != nil {
		ch <- state{result_flow.State_FAILED, err}
		return
	}

	for _, build := range builds {
		runner, err := transform.LoadTestRunnerBuild(ctx, buildIDMap[build.Id], build, c.source.GetBb())
		if err != nil {
			logging.Errorf(ctx, "failed to extract data from build: %v", err)
			continue
		}
		if err = c.bqTestRunClient.Insert(ctx, toTestRunRow(runner)); err != nil {
			logging.Errorf(ctx, "failed to upload TestRun data to Bigquery: %v", err)
		}
		if err = c.bqTestCaseClient.Insert(ctx, toTestCaseRows(runner)...); err != nil {
			logging.Errorf(ctx, "failed to upload TestCaseResult data to Bigquery: %v", err)
		}
	}
	c.bqTestRunClient.CloseAndDrain(ctx)
	c.bqTestCaseClient.CloseAndDrain(ctx)
	if err = c.mClient.AckMessages(ctx, msgs); err != nil {
		ch <- state{result_flow.State_FAILED, err}
		return
	}
	ch <- state{result_flow.State_SUCCEEDED, nil}
}

func (c *pipeTestRunnerDataRun) loadTestRunnerRequest() error {
	var (
		r   result_flow.TestRunnerRequest
		err error
	)
	if err = readJSONPb(c.inputPath, &r); err != nil {
		return err
	}
	if c.source, err = verifySource(r.GetTestRunner()); err != nil {
		return err
	}
	if c.testRun, err = verifyTarget(r.GetTestRun()); err != nil {
		return err
	}
	if c.testCaseResult, err = verifyTarget(r.GetTestCase()); err != nil {
		return err
	}
	c.deadline = getDeadline(r.GetDeadline(), site.DefaultDeadlineSeconds)
	return nil
}

func getBuildIDs(m map[int64]string) []int64 {
	var res []int64
	for b := range m {
		res = append(res, b)
	}
	return res
}

func toTestRunRow(b transform.TestRunnerBuild) bigquery.ValueSaver {
	row := b.ToTestRun()
	return &lucibq.Row{Message: row, InsertID: fmt.Sprintf("%d", row.GetBuildId())}
}

func toTestCaseRows(b transform.TestRunnerBuild) []bigquery.ValueSaver {
	var rows []bigquery.ValueSaver
	for _, t := range b.ToTestCaseResults() {
		rows = append(rows, &lucibq.Row{Message: t, InsertID: t.GetUid()})
	}
	return rows
}
