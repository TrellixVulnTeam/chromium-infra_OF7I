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
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"infra/chromeperf/pinpoint/proto"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/hardcoded/chromeinfra"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type baseCommandRun struct {
	subcommands.CommandRunBase
	endpoint string
	workDir  string

	json       bool
	jsonIndent string
	jsonPrefix string

	clientFactory clientFactory
}

func (r *baseCommandRun) RegisterFlags(p Param) userConfig {
	uc := getUserConfig(context.Background(), getUserConfigFilename(), p)
	r.Flags.StringVar(&r.endpoint, "endpoint", uc.Endpoint, text.Doc(`
		Pinpoint API service endpoint.
	`))
	r.Flags.StringVar(&r.workDir, "work-dir", uc.WorkDir, text.Doc(`
		Working directory for the tool when downloading files.
	`))
	r.Flags.BoolVar(&r.json, "json", false, text.Doc(`
		Optional json output format.
	`))
	r.Flags.StringVar(&r.jsonIndent, "json-indent", "", text.Doc(`
		Optional indent for json output format.
	`))
	r.Flags.StringVar(&r.jsonPrefix, "json-prefix", "", text.Doc(`
		Optional prefix for json output format.
	`))
	return uc
}

func (r *baseCommandRun) pinpointClient(ctx context.Context) (proto.PinpointClient, error) {
	r.clientFactory.init(ctx)

	endpoint := r.endpoint
	if !strings.Contains(endpoint, ":") {
		// If there is no port specified, assume we want gRPC's default.
		endpoint = fmt.Sprintf("%s:%d", endpoint, 443)
	}
	conn, err := r.clientFactory.grpc(endpoint)
	if err != nil {
		return nil, errors.Annotate(err, "failed dial grpc").Err()
	}
	return proto.NewPinpointClient(conn), nil
}

func (r *baseCommandRun) httpClient(ctx context.Context) (*http.Client, error) {
	r.clientFactory.init(ctx)

	return r.clientFactory.http()
}

func (r *baseCommandRun) writeJSON(out io.Writer, data interface{}) error {
	enc := json.NewEncoder(out)
	enc.SetIndent(r.jsonPrefix, r.jsonIndent)
	if err := enc.Encode(data); err != nil {
		return errors.Annotate(err, "could not render json").Err()
	}
	return nil
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

func removeExisting(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	if err := os.RemoveAll(path); err != nil {
		return errors.Annotate(err, "failed removing: %s", path).Err()
	}
	return nil
}

// clientFactory encapsulates the dialing and caching of grpc
// backends and http transports.
type clientFactory struct {
	tlsCreds    credentials.TransportCredentials
	baseAuth    *auth.Authenticator
	idTokenAuth *auth.Authenticator

	grpcConns singleflight.Group
	initOnce  sync.Once
}

func (f *clientFactory) init(ctx context.Context) {
	f.initOnce.Do(func() {
		f.tlsCreds = credentials.NewTLS(nil)

		opts := chromeinfra.DefaultAuthOptions()
		f.baseAuth = auth.NewAuthenticator(ctx, auth.InteractiveLogin, opts)

		opts.UseIDTokens = true
		f.idTokenAuth = auth.NewAuthenticator(ctx, auth.InteractiveLogin, opts)
	})
}

func (f *clientFactory) grpc(endpoint string) (*grpc.ClientConn, error) {
	conn, err, _ := f.grpcConns.Do(endpoint, func() (interface{}, error) {
		cred, err := f.idTokenAuth.PerRPCCredentials()
		if err != nil {
			return nil, errors.Annotate(err, "failed get per rpc credentials from luci auth").Err()
		}
		return grpc.Dial(
			endpoint,
			grpc.WithTransportCredentials(f.tlsCreds),
			grpc.WithPerRPCCredentials(cred),
		)
	})
	if err != nil {
		return nil, errors.Annotate(err, "failed get grpc conn").Err()
	}
	return conn.(*grpc.ClientConn), nil
}

func (f *clientFactory) http() (*http.Client, error) {
	return f.baseAuth.Client()
}
