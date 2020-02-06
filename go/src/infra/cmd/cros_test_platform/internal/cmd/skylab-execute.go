// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/maruel/subcommands"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/isolatedclient"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/proto/google"

	"infra/cmd/cros_test_platform/internal/execution/isolate"
	"infra/cmd/cros_test_platform/internal/execution/isolate/getter"
	"infra/cmd/cros_test_platform/internal/execution/skylab"
	"infra/libs/skylab/common/errctx"
	"infra/libs/skylab/swarming"
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

	d, err := inferDeadline(&request)
	if err != nil {
		return err
	}

	runner, err := skylab.NewRunner(cfg.SkylabWorker, taskID, d, request.TaggedRequests)
	if err != nil {
		return err
	}

	resps, err := c.handleRequests(ctx, d, runner, client, gf)
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

func containsSomeResponse(rs map[string]*steps.ExecuteResponse) bool {
	for _, r := range rs {
		if r != nil {
			return true
		}
	}
	return false
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
func (c *skylabExecuteRun) handleRequests(ctx context.Context, deadline time.Time, runner *skylab.Runner, t *swarming.Client, gf isolate.GetterFactory) (map[string]*steps.ExecuteResponse, error) {
	ctx, cancel := errctx.WithDeadline(ctx, deadline, fmt.Errorf("hit cros_test_platform request deadline (%s)", deadline))
	defer cancel(context.Canceled)
	err := runner.LaunchAndWait(ctx, skylab.Clients{
		Swarming:      t,
		IsolateGetter: gf,
	})
	return runner.Responses(), err
}

func (c *skylabExecuteRun) writeResponsesWithError(resps map[string]*steps.ExecuteResponse, err error) error {
	r := &steps.ExecuteResponses{TaggedResponses: resps}
	return writeResponseWithError(c.outputPath, r, err)
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

func httpClient(ctx context.Context, authJSONPath string) (*http.Client, error) {
	// TODO(akeshet): Specify ClientID and ClientSecret fields.
	options := auth.Options{
		ServiceAccountJSONPath: authJSONPath,
		Scopes:                 []string{auth.OAuthScopeEmail},
	}
	a := auth.NewAuthenticator(ctx, auth.SilentLogin, options)
	h, err := a.Client()
	if err != nil {
		return nil, errors.Annotate(err, "create http client").Err()
	}
	return h, nil
}

func swarmingClient(ctx context.Context, c *config.Config_Swarming) (*swarming.Client, error) {
	logging.Debugf(ctx, "Creating swarming client from config %v", c)
	hClient, err := httpClient(ctx, c.AuthJsonPath)
	if err != nil {
		return nil, err
	}

	client, err := swarming.New(ctx, hClient, c.Server)
	if err != nil {
		return nil, err
	}

	return client, nil
}
