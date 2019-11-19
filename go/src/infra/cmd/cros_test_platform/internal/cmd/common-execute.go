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
}

func (c *commonExecuteRun) addFlags() {
	c.Flags.StringVar(&c.inputPath, "input_json", "", "Path to JSON ExecuteRequests to read.")
	c.Flags.StringVar(&c.outputPath, "output_json", "", "Path to JSON ExecuteResponses to write.")
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
	return rs.Requests, nil
}

func (c *commonExecuteRun) writeResponsesWithError(resps []*steps.ExecuteResponse, err error) error {
	return writeResponseWithError(
		c.outputPath,
		&steps.ExecuteResponses{
			Responses: resps,
		},
		err,
	)
}

func (c *commonExecuteRun) handleRequests(ctx context.Context, maximumDuration time.Duration, runner execution.Runner, t *swarming.Client, gf isolate.GetterFactory) ([]*steps.ExecuteResponse, error) {
	ctx, cancel := errctx.WithTimeout(ctx, maximumDuration, fmt.Errorf("exceeded request's maximum duration"))
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
