// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/maruel/subcommands"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/proto/google"

	"infra/cmd/cros_test_platform/internal/execution"
	"infra/cmd/cros_test_platform/internal/execution/skylab"
	"infra/cmd/cros_test_platform/internal/execution/swarming"
	"infra/libs/skylab/common/errctx"
)

// SkylabExecute subcommand: Run a set of enumerated tests against skylab backend.
var SkylabExecute = &subcommands.Command{
	UsageLine: "skylab-execute -input_json /path/to/input.json -output_json /path/to/output.json",
	ShortDesc: "Run a set of enumerated tests against skylab backend.",
	LongDesc:  `Run a set of enumerated tests against skylab backend.`,
	CommandRun: func() subcommands.CommandRun {
		c := &skylabExecuteRun{}
		c.Flags.StringVar(&c.inputPath, "input_json", "", "Path to JSON ExecuteRequests to read.")
		c.Flags.StringVar(&c.outputPath, "output_json", "", "Path to JSON ExecuteResponses to write.")
		return c
	},
}

type skylabExecuteRun struct {
	subcommands.CommandRunBase
	inputPath  string
	outputPath string
}

func (c *skylabExecuteRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.validateArgs(); err != nil {
		fmt.Fprintln(a.GetErr(), err.Error())
		c.Flags.Usage()
		return exitCode(err)
	}

	err := c.innerRun(a, args, env)
	if err != nil {
		fmt.Fprintf(a.GetErr(), "%s\n", err)
	}
	return exitCode(err)
}

func (c *skylabExecuteRun) validateArgs() error {
	if c.inputPath == "" {
		return fmt.Errorf("-input_json not specified")
	}

	if c.outputPath == "" {
		return fmt.Errorf("-output_json not specified")
	}

	return nil
}

func (c *skylabExecuteRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ctx = setupLogging(ctx)

	var request steps.ExecuteRequests
	if err := readRequest(c.inputPath, &request); err != nil {
		return err
	}

	if err := c.validateRequests(request.TaggedRequests); err != nil {
		return err
	}

	cfg := extractOneConfig(request.TaggedRequests)
	client, err := swarming.NewSkylabClient(ctx, cfg)
	if err != nil {
		return err
	}

	d, err := inferDeadline(&request)
	if err != nil {
		return err
	}

	runner, err := execution.NewRunner(cfg.SkylabWorker, env["SWARMING_TASK_ID"].Value, d, request.TaggedRequests)
	if err != nil {
		return err
	}

	resps, err := c.handleRequests(ctx, d, runner, client)
	if err != nil {
		return err
	}
	c.updateWithEnumerationErrors(ctx, resps, request.TaggedRequests)
	return writeResponse(
		c.outputPath,
		&steps.ExecuteResponses{
			TaggedResponses: resps,
		},
	)
}

func extractOneConfig(trs map[string]*steps.ExecuteRequest) *config.Config {
	for _, r := range trs {
		return r.Config
	}
	return nil
}

func inferDeadline(r *steps.ExecuteRequests) (time.Time, error) {
	c := google.TimeFromProto(r.GetBuild().GetCreateTime())
	if c.IsZero() {
		return c, errors.Reason("infer deadline: build creation time not known").Err()
	}
	return c.Add(inferTimeout(r.TaggedRequests)), nil
}

const defaultTaskTimout = 12 * time.Hour

func inferTimeout(trs map[string]*steps.ExecuteRequest) time.Duration {
	for _, r := range trs {
		if maxDuration, err := ptypes.Duration(r.RequestParams.Time.MaximumDuration); err == nil {
			return maxDuration
		}
		return defaultTaskTimout
	}
	return defaultTaskTimout
}

func (c *skylabExecuteRun) validateRequests(trs map[string]*steps.ExecuteRequest) error {
	if len(trs) == 0 {
		return errors.Reason("zero requests").Err()
	}

	for t, r := range trs {
		if err := c.validateRequest(r); err != nil {
			return errors.Annotate(err, "validate request %s", t).Err()
		}
	}

	var sTag string
	var sReq *steps.ExecuteRequest
	for t, r := range trs {
		sTag = t
		sReq = r
		break
	}

	sCfg := sReq.Config
	if err := c.validateRequestConfig(sCfg); err != nil {
		return errors.Annotate(err, "validate request %s", sTag).Err()
	}
	for t, r := range trs {
		o := r.Config
		if !proto.Equal(sCfg, o) {
			return errors.Reason("validate request: mistmatched config: %s[%#v] vs %s[%#v]", sTag, sCfg, t, o).Err()
		}
	}

	sTimeout := sReq.RequestParams.Time.MaximumDuration
	for t, r := range trs {
		o := r.RequestParams.Time.MaximumDuration
		if !proto.Equal(sTimeout, o) {
			return errors.Reason("validate request: per-request timeout support unimplemented: %s[%s] vs %s[%s]", sTag, sTimeout, t, o).Err()
		}
	}
	return nil
}

func (c *skylabExecuteRun) validateRequest(request *steps.ExecuteRequest) error {
	if request == nil {
		return fmt.Errorf("nil request")
	}
	if request.Config == nil {
		return fmt.Errorf("nil request.config")
	}
	return nil
}

func (c *skylabExecuteRun) validateRequestConfig(cfg *config.Config) error {
	if cfg.SkylabSwarming == nil {
		return fmt.Errorf("nil request.config.skylab_swarming")
	}
	if cfg.SkylabIsolate == nil {
		return fmt.Errorf("nil request.config.skylab_isolate")
	}
	if cfg.SkylabWorker == nil {
		return fmt.Errorf("nil request.config.skylab_worker")
	}
	return nil
}

func (c *skylabExecuteRun) handleRequests(ctx context.Context, deadline time.Time, runner *execution.Runner, skylab skylab.Client) (map[string]*steps.ExecuteResponse, error) {
	tErr, fErr := runWithDeadline(
		ctx,
		func(ctx context.Context) error {
			return runner.LaunchAndWait(ctx, skylab)
		},
		deadline,
	)
	if tErr != nil {
		// Timeout while waiting for tasks is not considered an Test Platform
		// infrastructure error because root cause is mostly related to fleet
		// capacity or long test runtimes.
		logging.Warningf(ctx, "Exited wait dut to timeout: %s", tErr)
		logging.Warningf(ctx, "Execution responses will contain test failures as a consequence of the timeout.")
	}
	return runner.Responses(), fErr
}

func (c *skylabExecuteRun) updateWithEnumerationErrors(ctx context.Context, resps map[string]*steps.ExecuteResponse, reqs map[string]*steps.ExecuteRequest) {
	for t, resp := range resps {
		req, ok := reqs[t]
		if !ok {
			panic(fmt.Sprintf("request for non-existent request for %s", t))
		}
		if es := req.GetEnumeration().GetErrorSummary(); es != "" {
			if resp.State == nil {
				resp.State = &test_platform.TaskState{}
			}
			resp.State.Verdict = test_platform.TaskState_VERDICT_FAILED
			logging.Infof(ctx, "Set request %s to VERDICT_FAILED because of enumeration error: %s", t, es)
		}
	}
}

var timeoutTag = errors.BoolTag{Key: errors.NewTagKey("this error occurred because of timeout")}

// runWithDeadline runs f() with the given deadline.
//
// In case of a timeout, the error is returned as timeoutError. All other errors
// are returned as fErr.
func runWithDeadline(ctx context.Context, f func(context.Context) error, deadline time.Time) (timeoutError error, fErr error) {
	tErr := fmt.Errorf("hit cros_test_platform request deadline (%s)", deadline)
	tErr = timeoutTag.Apply(tErr)
	ctx, cancel := errctx.WithDeadline(ctx, deadline, tErr)
	defer cancel(context.Canceled)

	ierr := f(ctx)
	if timeoutTag.In(ierr) {
		return ierr, nil
	}
	return nil, ierr
}
