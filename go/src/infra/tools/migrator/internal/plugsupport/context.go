// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"context"
	"net/http"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/common/system/signals"
	"go.chromium.org/luci/config/cfgclient"
)

// RootContext constructs the root process context.
//
// `ctx` is usually context.Background(). Must be called at most once, since it
// registers signal handlers.
func RootContext(ctx context.Context) context.Context {
	ctx = gologger.StdConfig.Use(ctx)
	ctx = logging.SetLevel(ctx, logging.Info)
	ctx, cancel := context.WithCancel(ctx)
	signals.HandleInterrupt(cancel)
	return ctx
}

// ContextConfig contains serializable parameters for setting up a base context.
//
// They are passed across processes to (presumably) get similar contexts across
// them.
//
// We assume auth.Options are JSON-serializable. This is not guaranteed and
// stuff may break in the future. If this happens, a more advanced approach
// would be to use the `authctx` package to setup an authentication context
// in the `migrator` binary, and use it in the plugin subprocess.
type ContextConfig struct {
	Logging           logging.Config
	Auth              auth.Options
	ConfigServiceHost string
}

// Apply applies the config to the root context.
//
// Always returns a non-nil context, even on errors.
func (r *ContextConfig) Apply(ctx context.Context) (context.Context, error) {
	ctx = r.Logging.Set(ctx)

	authenticator := auth.NewAuthenticator(ctx, auth.SilentLogin, r.Auth)

	client, err := cfgclient.New(cfgclient.Options{
		ServiceHost: r.ConfigServiceHost,
		ClientFactory: func(context.Context) (*http.Client, error) {
			return authenticator.Client()
		},
	})
	if err != nil {
		return ctx, errors.Annotate(err, "cannot configure LUCI Config client").Err()
	}

	ctx = cfgclient.Use(ctx, client)
	return ctx, nil
}
