// Copyright 2021 The Chromium Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"context"
	"flag"
	"fmt"
	"infra/chromeperf/pinpoint"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc/credentials"
)

type baseCommandRun struct {
	subcommands.CommandRunBase
	endpoint string

	// Used in method pinpointClient to lazy-init the factory.
	initClientFactory     sync.Once
	pinpointClientFactory *pinpointClientFactory
	initClientFactoryErr  error
}

func (r *baseCommandRun) RegisterFlags(p Param) {
	r.Flags.StringVar(&r.endpoint, "endpoint", p.DefaultServiceDomain, text.Doc(`
		Pinpoint API service endpoint.
	`))
}

func getFactorySettings(ctx context.Context, serviceDomain string) (
	*tokenCache, credentials.TransportCredentials, error) {

	// If we are connecting to a local server, heuristically guess that we want
	// an insecure connection and return nil for all return values (which the
	// client factory takes that as an indication to use insecure).
	if strings.HasPrefix(serviceDomain, "localhost:") {
		return nil, nil, nil
	}

	cacheDir, found := os.LookupEnv("PINPOINT_CACHE_DIR")
	if !found {
		homeEnv, err := os.UserHomeDir()
		if err != nil {
			homeEnv = os.TempDir()
		}
		cacheDir = filepath.Join(homeEnv, ".cache", "pinpoint-cli")
	}
	tCache, err := newTokenCache(ctx, cacheDir)
	if err != nil {
		return nil, nil, errors.Annotate(err, "failed to create token cache").Err()
	}
	tlsCreds := credentials.NewTLS(nil)
	return tCache, tlsCreds, nil
}

func (r *baseCommandRun) pinpointClient(ctx context.Context) (pinpoint.PinpointClient, error) {
	r.initClientFactory.Do(func() {
		// We're setting up the client factory here, so that the end commands
		// do the connection on-demand. If we ever need to support a scripting
		// interface where we allow multiple requests to be made by the runner
		// concurrently, then we're good with that scenario too.
		tCache, tlsCreds, err := getFactorySettings(ctx, r.endpoint)
		if err != nil {
			r.initClientFactoryErr = errors.Annotate(err, "failed to initialize connection factory").Err()
			return
		}
		endpoint := r.endpoint
		if !strings.Contains(endpoint, ":") {
			// If there is no port specified, assume we want gRPC's default.
			endpoint = fmt.Sprintf("%s:%d", endpoint, 443)
		}
		r.pinpointClientFactory = newPinpointClientFactory(endpoint, tCache, tlsCreds)
	})
	if r.initClientFactoryErr != nil {
		return nil, r.initClientFactoryErr
	}
	c, err := r.pinpointClientFactory.Client(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "failed to create a Pinpoint client").Err()
	}
	return c, nil
}

type pinpointCommand interface {
	Run(ctx context.Context, a subcommands.Application, args []string) error
	RegisterFlags(p Param)
	GetFlags() *flag.FlagSet
}

type wrappedPinpointCommand struct {
	delegate pinpointCommand
}

func (wpc wrappedPinpointCommand) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	// The subcommands.CommandRun parameter to GetContext is only used to check
	// to see if it implements ContextModificator. Since we aren't using that,
	// no need to jump through hoops to support it.
	ctx := cli.GetContext(a, nil, env)
	err := wpc.delegate.Run(ctx, a, args)
	if err == nil {
		return 0
	}
	fmt.Fprintf(a.GetErr(), "ERROR: %s\n", err)
	return 1
}

func (wpc wrappedPinpointCommand) GetFlags() *flag.FlagSet {
	return wpc.delegate.GetFlags()
}

func wrapCommand(p Param, newCmd func() pinpointCommand) func() subcommands.CommandRun {
	return func() subcommands.CommandRun {
		cmd := newCmd()
		cmd.RegisterFlags(p)
		return wrappedPinpointCommand{cmd}
	}
}
