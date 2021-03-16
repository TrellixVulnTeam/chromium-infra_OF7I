// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufspb

import (
	"context"
	"fmt"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/grpc/prpc"
)

// Option is used to config the UFS client to be created (e.g. service name,
// user agent, etc.)
type Option func(*config)

// NewClient creates a new Client instance to access UFS.
// Usage example:
// c, err:= NewClient(ctx, ServiceAccountJSONPath("/path/to/json"), UserAgent("agent/3.0.0"))
// It's the caller's responsibility to specify the namespace in metadata of an
// outgoing context in each RPC call.
func NewClient(ctx context.Context, o ...Option) (FleetClient, error) {
	c := &config{ufsService: "ufs.api.cr.dev"}
	for _, f := range o {
		f(c)
	}

	a := auth.NewAuthenticator(ctx, auth.SilentLogin, c.authOption)
	hc, err := a.Client()
	if err != nil {
		return nil, fmt.Errorf("new UFS client: could not establish HTTP client: %s", err)
	}
	return NewFleetPRPCClient(&prpc.Client{
		C:    hc,
		Host: c.ufsService,
		Options: &prpc.Options{
			UserAgent: c.userAgent,
		},
	}), nil
}

// ServiceName defines the service name of the client to request.
func ServiceName(n string) Option {
	return Option(func(c *config) { c.ufsService = n })
}

// ServiceAccountJSONPath is the path of the service account JSON file to auth
// the client.
func ServiceAccountJSONPath(p string) Option {
	return Option(func(c *config) {
		c.authOption = auth.Options{
			Method:                 auth.ServiceAccountMethod,
			ServiceAccountJSONPath: p,
		}
	})
}

// UserAgent is the user agent of the client, e.g. "fleet-tlw/3.0.0".
// UFS only supports a user agent with version not lower than "3.0.0".
func UserAgent(a string) Option {
	return Option(func(c *config) { c.userAgent = a })
}

type config struct {
	ufsService string
	authOption auth.Options
	userAgent  string
}
