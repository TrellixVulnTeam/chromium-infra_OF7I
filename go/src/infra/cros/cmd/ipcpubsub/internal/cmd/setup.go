// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
)

type setupRun struct {
	baseRun
	name string
}

// CmdSetup describes the subcommand flags for setting up a subscription
var CmdSetup = &subcommands.Command{
	UsageLine: "setup -project [PROJECT] -topic [TOPIC] -name [NAME]",
	ShortDesc: "set up a subscription",
	CommandRun: func() subcommands.CommandRun {
		c := &setupRun{}
		c.registerCommonFlags(&c.Flags)
		c.Flags.StringVar(&c.name, "name", "", "name of subscription: must be 3-255 characters, start with a letter, and composed of alphanumerics and -_.~+% only")
		return c
	},
}

func (c *setupRun) validateArgs(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) error {
	if c.topic == "" {
		return errors.Reason("topic name is required").Err()
	}
	if c.name == "" {
		return errors.Reason("subscription name is required").Err()
	}
	return nil
}

func (c *setupRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	err := c.innerRun(a, args, env)
	if err != nil {
		fmt.Fprintln(a.GetErr(), err.Error())
		return 1
	}
	fmt.Fprintf(a.GetErr(), "Created subscription %s", c.name)
	return 0
}

func (c *setupRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	if err := c.validateArgs(ctx, a, args, env); err != nil {
		c.Flags.Usage()
		return err
	}
	client, err := pubsub.NewClient(ctx, c.project)
	if err != nil {
		return err
	}
	topic, err := client.CreateTopic(ctx, c.topic)
	if err != nil {
		return err
	}
	cfg := pubsub.SubscriptionConfig{Topic: topic}
	_, err = client.CreateSubscription(ctx, c.name, cfg)
	if err != nil {
		return err
	}
	return nil
}
