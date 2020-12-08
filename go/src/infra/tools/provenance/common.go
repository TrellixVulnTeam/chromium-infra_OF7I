// Copyright 2020 The LUCI Authors.
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

package main

import (
	"context"
	"io/ioutil"
	"os"
	"strings"

	"github.com/maruel/subcommands"

	cloudkms "cloud.google.com/go/kms/apiv1"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

type commonFlags struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	keyPath   string
}

func (c *commonFlags) Init(authOpts auth.Options) {
	c.authFlags.Register(&c.Flags, authOpts)
}

func (c *commonFlags) Validate(args []string) error {
	if len(args) < 1 {
		return errors.New("positional arguments missing")
	}
	if len(args) > 1 {
		return errors.Reason("unexpected positional arguments %q; only keyPath is allowed", args[1:]).Err()
	}
	if err := validateCryptoKeysKMSPath(args[0]); err != nil {
		return err
	}
	c.keyPath = args[0]
	return nil
}

func (c *commonFlags) createAuthTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	parsedAuthOpts, err := c.authFlags.Options()
	if err != nil {
		return nil, err
	}
	a := auth.NewAuthenticator(ctx, auth.SilentLogin, parsedAuthOpts)
	if err := a.CheckLoginRequired(); err != nil {
		return nil, errors.Annotate(err, "please login with `luci-auth login`").Err()
	}
	return a.TokenSource()
}

func (c *commonFlags) makeKMSClient(ctx context.Context) *cloudkms.KeyManagementClient {
	// Set up service.
	authTS, err := c.createAuthTokenSource(ctx)
	if err != nil {
		logging.WithError(err).Errorf(ctx, "Error while creating Auth token.")
	}
	client, err := cloudkms.NewKeyManagementClient(ctx, option.WithTokenSource(authTS))
	if err != nil {
		logging.WithError(err).Errorf(ctx, "Error while creating KMS client.")
	}
	return client
}

func readInput(file string) ([]byte, error) {
	// Provides a way to pass input as stdin.
	if file == "-" {
		return ioutil.ReadAll(os.Stdin)
	}
	return ioutil.ReadFile(file)
}

func writeOutput(file string, data []byte) error {
	if file == "-" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return ioutil.WriteFile(file, data, 0664)
}

// cryptoKeysPathComponents are the path components necessary for API calls related to
// crypto keys.
//
// This structure represents the following path format:
// projects/.../locations/.../keyRings/.../cryptoKeys/...
var cryptoKeysPathComponents = []string{
	"projects",
	"locations",
	"keyRings",
	"cryptoKeys",
	"cryptoKeyVersions",
}

// validateCryptoKeysKMSPath validates a cloudkms path used for the API calls currently
// supported by this client.
//
// What this means is we only care about paths that look exactly like the ones
// constructed from kmsPathComponents.
// TODO(akashmukherjee): Use regex for the validation logic.
func validateCryptoKeysKMSPath(path string) error {
	if path == "" {
		return errors.Reason("path should not be empty.").Err()
	}
	if path[0] == '/' {
		path = path[1:]
	}
	components := strings.Split(path, "/")
	if len(components) < (len(cryptoKeysPathComponents)-1)*2 || len(components) > len(cryptoKeysPathComponents)*2 {
		return errors.Reason("path should have the form %s", strings.Join(cryptoKeysPathComponents, "/.../")+"/...").Err()
	}
	for i, c := range components {
		if i%2 == 1 {
			continue
		}
		expect := cryptoKeysPathComponents[i/2]
		if c != expect {
			return errors.Reason("expected component %d to be %s, got %s", i+1, expect, c).Err()
		}
	}
	return nil
}

type generateRun struct {
	commonFlags
	input  string
	output string
}

func (s *generateRun) Init(authOpts auth.Options) {
	s.commonFlags.Init(authOpts)
	s.Flags.StringVar(&s.input, "input", "", "Path to file with payload information.")
	s.Flags.StringVar(&s.output, "output", "", "Path to write provenance to (use '-' for stdout).")
}

func (s *generateRun) Validate(args []string) error {
	if err := s.commonFlags.Validate(args); err != nil {
		return err
	}
	if s.input == "" {
		return errors.New("input file is required")
	}
	if s.output == "" {
		return errors.New("output location is required")
	}
	return nil
}

func (s *generateRun) execute(ctx context.Context) error {
	service := s.makeKMSClient(ctx)
	// Read in input manifest.
	data, err := readInput(s.input)
	if err != nil {
		return err
	}

	result, err := generateProvenance(ctx, service, data, s.keyPath)
	if err != nil {
		return err
	}
	// Write output provenance to outfile.
	return writeOutput(s.output, result)
}

func (s *generateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, s, env)
	if err := s.Validate(args); err != nil {
		logging.WithError(err).Errorf(ctx, "Error while validating arguments")
		return 1
	}
	if err := s.execute(ctx); err != nil {
		logging.WithError(err).Errorf(ctx, "Error while executing command")
		return 1
	}
	return 0
}
