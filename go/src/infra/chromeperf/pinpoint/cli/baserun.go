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
	"fmt"
	"os"
	"path/filepath"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc/credentials"
)

type baseCommandRun struct {
	subcommands.CommandRunBase
	endpoint              string
	pinpointClientFactory *pinpointClientFactory
}

func (r *baseCommandRun) RegisterDefaultFlags(p Param) {
	r.Flags.StringVar(&r.endpoint, "endpoint", p.DefaultServiceDomain, text.Doc(`
		Pinpoint API service endpoint.
	`))
}

func getFactorySettings(ctx context.Context, serviceDomain string) (
	*tokenCache, credentials.TransportCredentials, error) {
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

func (r *baseCommandRun) initFactory(ctx context.Context) error {
	// We're setting up the client factory here, so that the end commands
	// do the connection on-demand. If we ever need to support a scripting
	// interface where we allow multiple requests to be made by the runner
	// concurrently, then we're good with that scenario too.
	tCache, tlsCreds, err := getFactorySettings(ctx, r.endpoint)
	if err != nil {
		return errors.Annotate(err, "failed to initialize connection factory").Err()
	}
	r.pinpointClientFactory = newPinpointClientFactory(fmt.Sprintf("%s:%d", r.endpoint, 443), tCache, tlsCreds)
	return nil
}
