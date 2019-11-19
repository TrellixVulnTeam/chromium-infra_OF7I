// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/maruel/subcommands"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/cmd/cros_test_platform/internal/execution"
	"infra/cmd/cros_test_platform/internal/execution/isolate"
	"infra/libs/skylab/common/errctx"
	"infra/libs/skylab/swarming"
)

type commonExecuteRun struct {
	subcommands.CommandRunBase
	inputPath  string
	outputPath string

	// TODO(crbug.com/1002941) Completely transition to tagged requests only, once
	// - recipe has transitioned to using tagged requests
	// - autotest-execute has been deleted (this just reduces the work required).
	tagged      bool
	orderedTags []string
}

func (c *commonExecuteRun) addFlags() {
	c.Flags.StringVar(&c.inputPath, "input_json", "", "Path to JSON ExecuteRequests to read.")
	c.Flags.StringVar(&c.outputPath, "output_json", "", "Path to JSON ExecuteResponses to write.")
	c.Flags.BoolVar(&c.tagged, "tagged", false, "Transitional flag to enable tagged requests and responses.")
}

func (c *commonExecuteRun) validateArgs() error {
	if c.inputPath == "" {
		return fmt.Errorf("-input_json not specified")
	}

	if c.outputPath == "" {
		return fmt.Errorf("-output_json not specified")
	}

	return nil
}

func (c *commonExecuteRun) validateRequestCommon(request *steps.ExecuteRequest) error {
	if request == nil {
		return fmt.Errorf("nil request")
	}

	if request.Config == nil {
		return fmt.Errorf("nil request.config")
	}

	return nil
}

func (c *commonExecuteRun) readRequests() ([]*steps.ExecuteRequest, error) {
	var rs steps.ExecuteRequests
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

func (c *commonExecuteRun) unzipTaggedRequests(trs map[string]*steps.ExecuteRequest) ([]string, []*steps.ExecuteRequest) {
	var ts []string
	var rs []*steps.ExecuteRequest
	for t, r := range trs {
		ts = append(ts, t)
		rs = append(rs, r)
	}
	return ts, rs
}

func (c *commonExecuteRun) writeResponsesWithError(resps []*steps.ExecuteResponse, err error) error {
	r := &steps.ExecuteResponses{
		Responses: resps,
	}
	if c.tagged {
		r.TaggedResponses = c.zipTaggedResponses(c.orderedTags, resps)
	}
	return writeResponseWithError(c.outputPath, r, err)
}

func (c *commonExecuteRun) zipTaggedResponses(ts []string, rs []*steps.ExecuteResponse) map[string]*steps.ExecuteResponse {
	if len(ts) != len(rs) {
		panic(fmt.Sprintf("got %d responses for %d tags (%s)", len(rs), len(ts), ts))
	}
	m := make(map[string]*steps.ExecuteResponse)
	for i := range ts {
		m[ts[i]] = rs[i]
	}
	return m
}

func (c *commonExecuteRun) handleRequests(ctx context.Context, maximumDuration time.Duration, runner execution.Runner, t *swarming.Client, gf isolate.GetterFactory) ([]*steps.ExecuteResponse, error) {
	ctx, cancel := errctx.WithTimeout(ctx, maximumDuration, fmt.Errorf("cros_test_platform request timeout (after %s)", maximumDuration))
	defer cancel(context.Canceled)
	err := runner.LaunchAndWait(ctx, t, gf)
	return runner.Responses(t), err
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

func containsSomeResponse(rs []*steps.ExecuteResponse) bool {
	for _, r := range rs {
		if r != nil {
			return true
		}
	}
	return false
}
