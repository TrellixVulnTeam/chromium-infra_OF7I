// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package client

import (
	"context"

	"go.chromium.org/luci/auth"
	prpc "go.chromium.org/luci/grpc/prpc"

	kartepb "infra/cros/karte/api"
	"infra/cros/karte/internal/errors"
	"infra/cros/karte/internal/site"
)

// Option is a configuration option. For example `UserAgent(...)` would be
// an option.
type Option func(*Config)

// Config stores options needed for the Karte service.
type Config struct {
	karteService string
	authOption   auth.Options
	loginMode    auth.LoginMode
	userAgent    string
}

// LocalConfig returns a configuration for the Karte client intended for local use,
// such as in the karte command line tool.
//
// The auth options are required and must be passed in explicitly. The hostname and
// type of login are set to reasonable defaults for a local command line tool.
func LocalConfig(o auth.Options) *Config {
	return &Config{
		karteService: "127.0.0.1:8800",
		loginMode:    auth.InteractiveLogin,
		authOption:   o,
		userAgent:    "local command line tool",
	}
}

// DevConfig returns a configuration for the Karte client intended to talk to the dev
// instance of Karte, which is a cloud project.
//
// The auth options are required and must be passed in explicitly. The hostname and
// type of login are set to reasonable defaults for a local command line tool.
func DevConfig(o auth.Options) *Config {
	return &Config{
		karteService: site.DevKarteServer,
		loginMode:    auth.InteractiveLogin,
		authOption:   o,
		userAgent:    "local command line tool",
	}
}

// ProdConfig returns a configuration for the Karte client intended to talk to the dev
// instance of Karte, which is a cloud project.
//
// The auth options are required and must be passed in explicitly. The hostname and
// type of login are set to reasonable defaults for a local command line tool.
func ProdConfig(o auth.Options) *Config {
	return &Config{
		karteService: site.ProdKarteServer,
		loginMode:    auth.InteractiveLogin,
		authOption:   o,
		userAgent:    "local command line tool",
	}
}

// EmptyConfig is a config with no content. It is expected to fail to construct a client if used as the
// base config without the appropriate options being set.
func EmptyConfig() *Config {
	return nil
}

// NewClient creates a new client for the Karte service.
func NewClient(ctx context.Context, c *Config, o ...Option) (kartepb.KarteClient, error) {
	if c == nil {
		return nil, errors.New("karte client: cannot create new client from empty base config")
	}
	for _, f := range o {
		f(c)
	}

	a := auth.NewAuthenticator(ctx, c.loginMode, c.authOption)
	hc, err := a.Client()
	if err != nil {
		return nil, errors.Annotate(err, "create karte client").Err()
	}
	return kartepb.NewKartePRPCClient(&prpc.Client{
		C:    hc,
		Host: c.karteService,
		Options: &prpc.Options{
			UserAgent: c.userAgent,
		},
	}), nil
}

// NewCronClient creates a new client for the Karte service.
func NewCronClient(ctx context.Context, c *Config, o ...Option) (kartepb.KarteCronClient, error) {
	if c == nil {
		return nil, errors.New("karte cron client: cannot create new client from empty base config")
	}
	for _, f := range o {
		f(c)
	}

	a := auth.NewAuthenticator(ctx, c.loginMode, c.authOption)
	hc, err := a.Client()
	if err != nil {
		return nil, errors.Annotate(err, "create karte cron client").Err()
	}
	return kartepb.NewKarteCronPRPCClient(&prpc.Client{
		C:    hc,
		Host: c.karteService,
		Options: &prpc.Options{
			UserAgent: c.userAgent,
		},
	}), nil
}
