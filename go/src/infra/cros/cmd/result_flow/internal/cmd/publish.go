// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/proto/google"
	"google.golang.org/api/option"

	"infra/cros/cmd/result_flow/internal/message"
	"infra/cros/cmd/result_flow/internal/site"
)

// Publish subcommand pushes a build ID to Pub/Sub topic.
var Publish = &subcommands.Command{
	UsageLine: `publish [FLAGS...]`,
	ShortDesc: "Publish a Build ID",
	LongDesc: `Publish a Build ID to assigned Pub/Sub topic.
The build ID is stored in the message attribute having "build-id" as the key, same
with Buildbucket's Pub/Sub notification message.`,
	CommandRun: func() subcommands.CommandRun {
		c := &publishRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)

		c.Flags.StringVar(&c.inputPath, "input_json", "", "Path that contains JSON encoded test_platform.result_flow.PublishRequest")
		c.Flags.StringVar(&c.outputPath, "output_json", "", "Path to write JSON encoded test_platform.result_flow.PublishResponse to")
		return c
	},
}

type publishRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	inputPath  string
	outputPath string

	buildID  int64
	deadline time.Time

	config *result_flow.PubSubConfig

	clientOpts option.ClientOption
}

func (c *publishRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), err.Error())
		return 1
	}
	return 0
}

func (c *publishRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	var err error
	if err := c.loadPublishRequest(); err != nil {
		return err
	}
	ctx := context.Background()

	authOpts, err := c.authFlags.Options()
	if err != nil {
		return err
	}
	if c.clientOpts, err = newGRPCClientOptions(ctx, authOpts); err != nil {
		return err
	}

	var cf context.CancelFunc
	logging.Infof(ctx, "Running with deadline %s (current time: %s)", c.deadline.UTC(), time.Now().UTC())
	ctx, cf = context.WithDeadline(ctx, c.deadline)
	defer cf()

	s, err := c.runWithDeadline(ctx)
	werr := writeJSONPb(c.outputPath, &result_flow.PublishResponse{State: s})
	if err != nil {
		return err
	}
	return werr
}

func (c *publishRun) runWithDeadline(ctx context.Context) (result_flow.State, error) {
	ch := make(chan state, 1)
	go c.pipelineRun(ctx, ch)
	select {
	case <-ctx.Done():
		return result_flow.State_TIMED_OUT, fmt.Errorf("publish command hit the deadline")
	case res := <-ch:
		return res.r, res.e
	}
}

func (c *publishRun) pipelineRun(ctx context.Context, ch chan state) {
	defer close(ch)

	if err := message.PublishBuildID(ctx, c.buildID, c.config, c.clientOpts); err != nil {
		ch <- state{
			result_flow.State_FAILED,
			errors.Annotate(err, "failed to publish build ID %d", c.buildID).Err(),
		}
		return
	}
	ch <- state{result_flow.State_SUCCEEDED, nil}
	return
}

func (c *publishRun) loadPublishRequest() error {
	var (
		r   result_flow.PublishRequest
		err error
	)
	if err = readJSONPb(c.inputPath, &r); err != nil {
		return err
	}
	if c.config, err = getPubSubConf(r); err != nil {
		return err
	}
	if c.buildID, err = getBuildID(r.GetBuildId()); err != nil {
		return err
	}
	if r.GetDeadline() != nil {
		c.deadline = google.TimeFromProto(r.GetDeadline())
	} else {
		c.deadline = time.Now().Add(time.Second * time.Duration(site.DefaultDeadlineSeconds))
	}
	return nil
}

func getBuildID(bID int64) (int64, error) {
	if bID == 0 {
		return 0, fmt.Errorf("Build ID should be set")
	}
	return bID, nil
}

func getPubSubConf(r result_flow.PublishRequest) (*result_flow.PubSubConfig, error) {
	var p *result_flow.PubSubConfig
	if r.GetCtp() != nil {
		p = r.GetCtp()
	}
	if r.GetTestRunner() != nil {
		p = r.GetTestRunner()
	}
	return verifyTopic(p)
}

func verifyTopic(p *result_flow.PubSubConfig) (*result_flow.PubSubConfig, error) {
	if p == nil {
		return nil, fmt.Errorf("Either CTP or Skylab source should be set")
	}
	if p.GetProject() == "" {
		return nil, fmt.Errorf("Pubsub project should be set")
	}
	if p.GetTopic() == "" {
		return nil, fmt.Errorf("Pubsub topic should be set")
	}
	return p, nil
}
