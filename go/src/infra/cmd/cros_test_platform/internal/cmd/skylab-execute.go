// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/maruel/subcommands"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/isolatedclient"

	"infra/cmd/cros_test_platform/internal/execution"
	"infra/cmd/cros_test_platform/internal/execution/isolate"
	"infra/cmd/cros_test_platform/internal/execution/isolate/getter"
)

// SkylabExecute subcommand: Run a set of enumerated tests against skylab backend.
var SkylabExecute = &subcommands.Command{
	UsageLine: "skylab-execute -input_json /path/to/input.json -output_json /path/to/output.json",
	ShortDesc: "Run a set of enumerated tests against skylab backend.",
	LongDesc:  `Run a set of enumerated tests against skylab backend.`,
	CommandRun: func() subcommands.CommandRun {
		c := &skylabExecuteRun{}
		c.addFlags()
		return c
	},
}

type skylabExecuteRun struct {
	commonExecuteRun
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

func (c *skylabExecuteRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ctx = setupLogging(ctx)

	requests, err := c.readRequests()
	if err != nil {
		return err
	}
	if err := c.validateRequests(requests); err != nil {
		return err
	}

	cfg := requests[0].Config
	client, err := swarmingClient(ctx, cfg.SkylabSwarming)
	if err != nil {
		return err
	}
	gf := c.getterFactory(cfg.SkylabIsolate)

	var taskID string
	// taskID will be used as the parent task ID for child jobs created by
	// this execution. This is only valid if the child runs on the same swarming
	// instance as the parent (which is not true for cros_test_platform-dev).
	// TODO(crbug.com/994289): Move cros_test_platform-dev to the same instance
	// as its child jobs, then delete this conditional.
	if sameHost(env["SWARMING_SERVER"].Value, cfg.SkylabSwarming.Server) {
		taskID = env["SWARMING_TASK_ID"].Value
	}

	runner, err := execution.NewSkylabRunner(ctx, cfg.SkylabWorker, taskID, requests)
	if err != nil {
		return err
	}
	maxDuration, err := ptypes.Duration(requests[0].RequestParams.Time.MaximumDuration)
	if err != nil {
		maxDuration = 12 * time.Hour
	}

	resps, err := c.handleRequests(ctx, maxDuration, runner, client, gf)
	if err != nil && !containsSomeResponse(resps) {
		// Catastrophic error. There is no reasonable response to write.
		return err
	}
	return c.writeResponsesWithError(resps, err)
}

func sameHost(urlA, urlB string) bool {
	a, err := url.Parse(urlA)
	if err != nil {
		return false
	}
	b, err := url.Parse(urlB)
	if err != nil {
		return false
	}
	return a.Host == b.Host
}

func (c *skylabExecuteRun) validateRequests(requests []*steps.ExecuteRequest) error {
	if len(requests) == 0 {
		return errors.Reason("zero requests").Err()
	}
	for _, r := range requests {
		if err := c.validateRequestCommon(r); err != nil {
			return errors.Annotate(err, "validate request %s", r).Err()
		}
	}
	if err := c.validateRequestConfig(requests[0].Config); err != nil {
		return errors.Annotate(err, "validate request %s", requests[0]).Err()
	}

	cfg := requests[0].Config
	for _, r := range requests[1:] {
		o := r.Config
		if !proto.Equal(cfg, o) {
			return errors.Reason("mistmatched config: %s vs %s", cfg, o).Err()
		}
	}
	timeout := requests[0].RequestParams.Time.MaximumDuration
	for _, r := range requests[1:] {
		o := r.RequestParams.Time.MaximumDuration
		if !proto.Equal(timeout, o) {
			return errors.Reason("per-request timeout support unimplemented: %s vs %s", timeout, o).Err()
		}
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

func (c *skylabExecuteRun) getterFactory(conf *config.Config_Isolate) isolate.GetterFactory {
	return func(ctx context.Context, server string) (isolate.Getter, error) {
		hClient, err := httpClient(ctx, conf.AuthJsonPath)
		if err != nil {
			return nil, err
		}

		isolateClient := isolatedclient.New(nil, hClient, server, isolatedclient.DefaultNamespace, nil, nil)

		return getter.New(isolateClient), nil
	}
}
