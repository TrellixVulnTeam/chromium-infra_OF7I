// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package grpcclient provides a common configuration for specifying
// clients of grpc services. It includes setup hooks for default interceptors
// like deadlines, retry  policies, auth etc as well as a common Options
// struct for identifying backends and their connection settings.
package grpcclient

import (
	"context"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/grpc"

	"go.chromium.org/luci/server/auth"
)

// Options defines how to connect to a service.
type Options struct {
	// Address is the endpoint address of the service.
	Address string
	// DefaultTimeoutMs is the default timeout for calls to this service.
	DefaultTimeoutMs int
	// TODO: Retry policies, throttling, custom auth settings etc.
}

func (c *Options) timeoutInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.DefaultTimeoutMs)*time.Millisecond)
	defer cancel()
	return invoker(ctx, method, req, reply, cc, opts...)
}

// DefaultClientOptions returns grpc client options used for creating new instances of
// clients.
// TODO: Add monitoring interceptor for tsmon, stackdriver, prometheus etc as needed.
// TODO: Add throttling.
func (c *Options) DefaultClientOptions(ctx context.Context) ([]option.ClientOption, error) {
	creds, err := auth.GetPerRPCCredentials(ctx, auth.AsSelf, auth.WithScopes(auth.CloudOAuthScopes...))
	if err != nil {
		return nil, err
	}

	return []option.ClientOption{
		option.WithEndpoint(c.Address),
		option.WithGRPCDialOption(
			grpc.WithPerRPCCredentials(creds),
		),
		option.WithGRPCDialOption(
			grpc.WithUnaryInterceptor(
				c.timeoutInterceptor,
			),
		),
	}, nil
}
